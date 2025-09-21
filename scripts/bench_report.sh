#!/usr/bin/env bash
set -euo pipefail

# bench_report.sh
# - Run goskema benches and comparison benches
# - Parse outputs and generate a Markdown summary table

ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
OUT_FILE="${OUT:-${ROOT_DIR}/BENCH_RESULTS.md}"
BENCHTIME="${BENCHTIME:-1x}"
FILTER="${BENCH_FILTER:-.}"
SMALL_BENCHTIME="${SMALL_BENCHTIME:-2s}"
HUGE_BENCHTIME="${HUGE_BENCHTIME:-1x}"

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

MAIN_TXT="${TMP_DIR}/bench_main.txt"
COMP_TXT="${TMP_DIR}/bench_compare.txt"
MERGED_TSV="${TMP_DIR}/bench_merged.tsv"

echo "[bench_report] Running main benches..." 1>&2
(
  cd "${ROOT_DIR}" && \
  ( \
    go test ${GO_TAGS:+-tags ${GO_TAGS}} -run ^$ -bench "Parse.*_Small$" -benchmem -benchtime="${SMALL_BENCHTIME}" ./benchmarks; \
    go test ${GO_TAGS:+-tags ${GO_TAGS}} -run ^$ -bench "Parse.*_HugeArray$" -benchmem -benchtime="${HUGE_BENCHTIME}" ./benchmarks; \
    # compiled vs interpreted (examples/user)
    go test ${GO_TAGS:+-tags ${GO_TAGS}} -run ^$ -bench "(Compiled|Interpreted).*_Small" -benchmem -benchtime="${SMALL_BENCHTIME}" ./benchmarks; \
    true \
  ) | tee "${MAIN_TXT}"
)

echo "[bench_report] Running compare benches..." 1>&2
(
  cd "${ROOT_DIR}/benchmarks/compare" && \
  go mod tidy >/dev/null 2>&1 || true && \
  ( \
    go test ${GO_TAGS:+-tags ${GO_TAGS}} -run ^$ -bench "^Benchmark_.*_Small$" -benchmem -benchtime="${SMALL_BENCHTIME}"; \
    go test ${GO_TAGS:+-tags ${GO_TAGS}} -run ^$ -bench "^Benchmark_.*_HugeArray$" -benchmem -benchtime="${HUGE_BENCHTIME}"; \
    true \
  ) | tee "${COMP_TXT}"
)

# Extract CPU line if present (from compare run)
CPU_LINE=$(grep -m1 '^cpu:' "${COMP_TXT}" || true)
GOOS=$(go env GOOS)
GOARCH=$(go env GOARCH)
GOVER=$(go version | awk '{print $3, $4, $5}')

parse_to_tsv() {
  local suite="$1"; shift
  local file="$1"; shift
  awk -v SUITE="$suite" '
    BEGIN { FS=" "; OFS="\t" }
    /^Benchmark/ {
      name=$1
      ns=""; mbps=""; bop=""; allocs=""
      for (i=1;i<=NF;i++) {
        if ($i=="ns/op" && i>1) ns=$(i-1)
        if ($i=="MB/s" && i>1) mbps=$(i-1)
        if ($i=="B/op" && i>1) bop=$(i-1)
        if ($i=="allocs/op" && i>1) allocs=$(i-1)
      }
      grp="Misc"
      if (name ~ /^Benchmark_ParseOnly_.*_Small$/) grp="ParseOnly/Small"
      else if (name ~ /^Benchmark_ParseOnly_.*_HugeArray$/) grp="ParseOnly/HugeArray"
      else if (name ~ /^Benchmark_ParseAndCheck_.*_Small$/) grp="ParseAndCheck/Small"
      else if (name ~ /^Benchmark_ParseAndCheck_.*_HugeArray$/) grp="ParseAndCheck/HugeArray"
      else if (name ~ /^Benchmark_ParseAndValidateSchema_.*_Small$/) grp="ParseAndValidateSchema/Small"
      else if (name ~ /^Benchmark_(Compiled|Interpreted)_User_.*_Small/) grp="CompiledVsInterpreted/Small"
      else if (name ~ /_Small/) grp="Small"
      else if (name ~ /_HugeArray/) grp="HugeArray"
      else if (name ~ /Huge/) grp="Huge"
      print SUITE, grp, name, ns, mbps, bop, allocs
    }
  ' "$file"
}

{
  parse_to_tsv main "${MAIN_TXT}"
  parse_to_tsv compare "${COMP_TXT}"
} > "${MERGED_TSV}"

# Render Markdown tables grouped by suite/group with relative speed (ns/op)
render_md() {
  local tsv="$1"
  awk -F"\t" '
    function fmt(x){ if(x=="")return "-"; return x }
    function relmark(r){ if(r=="-")return "-"; if(r<=1.05)return "🏆 x" r; if(r<=2.0)return "✅ x" r; return "🐢 x" r }
    # derive library label from benchmark name
    function libof(name,   lib){
      if (name ~ /^Benchmark_encodingJSON_/) lib="encoding/json";
      else if (name ~ /^Benchmark_stdlib_/) lib="encoding/json";
      else if (name ~ /^Benchmark_gojson_/) lib="go-json";
      else if (name ~ /^Benchmark_jsoniter_/) lib="json-iterator";
      else if (name ~ /^Benchmark_sonic_/) lib="sonic";
      else if (name ~ /^Benchmark_fastjson_/) lib="fastjson";
      else if (name ~ /^Benchmark_jsonschema_v5_/) lib="jsonschema/v5";
      else if (name ~ /^Benchmark_goskema_/) lib="goskema";
      else if (name ~ /^Benchmark_ParseOnly_stdlib_/) lib="encoding/json";
      else if (name ~ /^Benchmark_ParseOnly_gojson_/) lib="go-json";
      else if (name ~ /^Benchmark_ParseOnly_jsoniter_/) lib="json-iterator";
      else if (name ~ /^Benchmark_ParseOnly_sonic_/) lib="sonic";
      else if (name ~ /^Benchmark_ParseOnly_fastjson_/) lib="fastjson";
      else if (name ~ /^Benchmark_ParseOnly_goskema_/) lib="goskema";
      else if (name ~ /^Benchmark_ParseAndCheck_goskema_/) lib="goskema";
      else if (name ~ /^Benchmark_ParseAndCheck_stdlib_/) lib="encoding/json";
      else if (name ~ /^Benchmark_ParseAndValidateSchema_jsonschema_v5_/) lib="jsonschema/v5";
      else if (name ~ /^Benchmark_ParseAndValidateSchema_goskema_/) lib="goskema";
      else if (name ~ /^Benchmark_ParseFromWithMeta_/) lib="goskema";
      else if (name ~ /^Benchmark_ParseFrom_/) lib="goskema";
      else if (name ~ /^Benchmark_StreamParse_/) lib="goskema";
      else if (name ~ /^Benchmark_Compiled_/) lib="compiled";
      else if (name ~ /^Benchmark_Interpreted_/) lib="interpreted";
      else lib="unknown";
      return lib;
    }
    function group_weight(g){
      if (g ~ /^ParseOnly\//) return 0
      if (g ~ /^ParseAndCheck\//) return 1
      if (g ~ /^ParseAndValidateSchema\//) return 2
      if (g ~ /^CompiledVsInterpreted\//) return 3
      return 9
    }
    function compare_keys(i1, v1, i2, v2,   a1,a2) {
      split(i1,a1,"|"); split(i2,a2,"|")
      prio1=(a1[1]=="compare"?0:1)
      prio2=(a2[1]=="compare"?0:1)
      gw1=group_weight(a1[2]); gw2=group_weight(a2[2])
      if (gw1!=gw2) return gw1 < gw2
      if (prio1!=prio2) return prio1 < prio2
      if (a1[2]!=a2[2]) return a1[2] < a2[2]
      return i1 < i2
    }
    {
      suite=$1; grp=$2; name=$3; ns=$4; mbps=$5; bop=$6; allocs=$7
      key=suite"|"grp
      if (!(key in minns) || (ns+0) < (minns[key]+0)) minns[key]=ns
      rows[key,++cnt[key]]=suite"\t"grp"\t"name"\t"ns"\t"mbps"\t"bop"\t"allocs
      keys[key]=1
    }
    END {
      print "## Benchmark Report"
      print ""
      for (k in keys) kk[++nkeys]=k
      # sort groups: compare|Small, compare|HugeArray, main|*
      PROCINFO["sorted_in"] = "compare_keys"
      for (k in keys) {
        split(k, p, "|")
        suite=p[1]; grp=p[2]
        print "### " suite " / " grp
        print ""
        print "| Lib | Name | ns/op | MB/s | B/op | allocs/op | Rel |"
        print "|---|---|---:|---:|---:|---:|---:|"
        # collect rows and sort by ns/op asc
        n=cnt[k]
        for (i=1;i<=n;i++) arr[i]=rows[k,i]
        # simple bubble sort due to awk portability
        for (i=1;i<=n;i++) for (j=i+1;j<=n;j++) {
          split(arr[i],ri,"\t"); split(arr[j],rj,"\t")
          if ((ri[4]+0) > (rj[4]+0)) { tmp=arr[i]; arr[i]=arr[j]; arr[j]=tmp }
        }
        for (i=1;i<=n;i++) {
          split(arr[i],r,"\t")
          suite=r[1]; grp=r[2]; name=r[3]; ns=r[4]; mbps=r[5]; bop=r[6]; allocs=r[7]
          base=minns[k]
          rel="-"
          if (ns!="" && base!="" && base+0>0) rel=sprintf("%.2f", (ns+0)/(base+0))
          printf("| %s | %s | %s | %s | %s | %s | %s |\n", libof(name), name, fmt(ns), fmt(mbps), fmt(bop), fmt(allocs), relmark(rel))
        }
        print ""
      }
    }
  ' "$tsv"
}

# Write final report
{
  echo "# Goskema Benchmarks"
  echo ""
  echo "- Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  echo "- Go: ${GOVER} (${GOOS}/${GOARCH})"
  if [[ -n "${CPU_LINE}" ]]; then echo "- ${CPU_LINE}"; fi
  echo ""
  render_md "${MERGED_TSV}"
} > "${OUT_FILE}"

echo "[bench_report] Wrote ${OUT_FILE}" 1>&2


