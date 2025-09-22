package i18n

// Translator retrieves localized messages for Issue codes.
// data provides optional metadata to embed in the message (for example,
// "expected" or "key").
type Translator interface {
	Message(code string, data map[string]string) string
}

// dictTranslator is the built-in dictionary-based Translator.
type dictTranslator struct{ lang string }

func (t dictTranslator) Message(code string, data map[string]string) string {
	switch t.lang {
	case "ja":
		switch code {
		case "invalid_type":
			return "型が不正です"
		case "required":
			return "必須プロパティが不足しています"
		case "unknown_key":
			return "未知のキーです"
		case "duplicate_key":
			return "キーが重複しています"
		case "too_small":
			return "値が小さすぎます"
		case "too_big":
			return "値が大きすぎます"
		case "too_short":
			return "短すぎます"
		case "too_long":
			return "長すぎます"
		case "pattern":
			return "パターンに一致しません"
		case "invalid_enum":
			return "列挙値が不正です"
		case "invalid_format":
			return "フォーマットが不正です"
		case "discriminator_missing":
			return "識別子(discriminator)が不足しています"
		case "discriminator_unknown":
			return "不明な識別子(discriminator)です"
		case "union_ambiguous":
			return "Unionの解決が曖昧です"
		case "parse_error":
			return "解析エラー"
		case "truncated":
			return "打ち切られました"
		case "overflow":
			return "オーバーフローしました"
		case "domain_range":
			return "範囲外です"
		case "aggregate_violation":
			return "集合制約に違反しています"
		case "uniqueness":
			return "値が重複しています"
		case "business_rule":
			return "業務ルールに違反しています"
		case "conflict":
			return "競合が発生しました"
		case "dependency_unavailable":
			return "依存先サービスが利用できません"
		}
	default: // "en"
		switch code {
		case "invalid_type":
			return "invalid type"
		case "required":
			return "required property missing"
		case "unknown_key":
			return "unknown key"
		case "duplicate_key":
			return "duplicate key"
		case "too_small":
			return "too small"
		case "too_big":
			return "too big"
		case "too_short":
			return "too short"
		case "too_long":
			return "too long"
		case "pattern":
			return "pattern mismatch"
		case "invalid_enum":
			return "invalid enum value"
		case "invalid_format":
			return "invalid format"
		case "discriminator_missing":
			return "discriminator missing"
		case "discriminator_unknown":
			return "discriminator unknown"
		case "union_ambiguous":
			return "union ambiguous"
		case "parse_error":
			return "parse error"
		case "truncated":
			return "truncated"
		case "overflow":
			return "overflow"
		case "domain_range":
			return "out of range"
		case "aggregate_violation":
			return "aggregate violation"
		case "uniqueness":
			return "duplicate value"
		case "business_rule":
			return "business rule violation"
		case "conflict":
			return "conflict"
		case "dependency_unavailable":
			return "dependency unavailable"
		}
	}
	return code
}

var currentTranslator Translator = dictTranslator{lang: "en"}

// SetLanguage switches the built-in Translator language ("en"/"ja").
func SetLanguage(lang string) {
	if lang != "ja" {
		lang = "en"
	}
	currentTranslator = dictTranslator{lang: lang}
}

// SetTranslator replaces the Translator implementation (not limited to the
// dictionary version).
func SetTranslator(tr Translator) {
	if tr == nil {
		currentTranslator = dictTranslator{lang: "en"}
		return
	}
	currentTranslator = tr
}

// T fetches a message for the given code using the current Translator.
func T(code string, data map[string]string) string { return currentTranslator.Message(code, data) }
