package agents

import (
	"regexp"
	"strings"
)

// filterUnsafeOutput applies deterministic safety rules to agent output.
// This is a hard filter — not prompt-based — so it cannot be bypassed by
// prompt injection or model hallucination.
func filterUnsafeOutput(text string) string {
	text = filterDangerousDirectives(text)
	text = appendDisclaimerIfNeeded(text)
	return text
}

// dangerousDirective matches patterns where the AI tells the user to stop/start
// medications or change dosages without framing it as a doctor discussion point.
var dangerousDirectives = []*regexp.Regexp{
	// "stop taking X" / "discontinue X" without "ask your doctor" nearby
	regexp.MustCompile(`(?i)\b(stop\s+taking|discontinue|quit\s+taking|cease\s+taking)\s+\w+`),
	// "take X mg" / "increase to X mg" — direct dosage instructions
	regexp.MustCompile(`(?i)\b(take|increase\s+to|decrease\s+to|reduce\s+to|switch\s+to)\s+\d+\s*(mg|ml|mcg|units?|iu)\b`),
	// "you should start taking X"
	regexp.MustCompile(`(?i)\byou\s+should\s+(start|begin)\s+taking\s+\w+`),
}

// safetyQualifiers — if any of these appear near a dangerous directive,
// the directive is considered safely framed.
var safetyQualifiers = []string{
	"ask your doctor",
	"discuss with your doctor",
	"talk to your doctor",
	"consult your doctor",
	"speak with your doctor",
	"your doctor may",
	"your doctor might",
	"your provider",
	"with medical supervision",
	"under medical guidance",
	"before making any changes",
	"do not change",
	"don't change",
	"bring this up with",
	"mention this to",
	"question for your doctor",
}

func filterDangerousDirectives(text string) string {
	lower := strings.ToLower(text)

	for _, re := range dangerousDirectives {
		matches := re.FindAllStringIndex(lower, -1)
		if len(matches) == 0 {
			continue
		}

		for _, match := range matches {
			// Check if there's a safety qualifier within 200 chars of the match
			start := match[0] - 200
			if start < 0 {
				start = 0
			}
			end := match[1] + 200
			if end > len(lower) {
				end = len(lower)
			}
			window := lower[start:end]

			hasSafetyQualifier := false
			for _, q := range safetyQualifiers {
				if strings.Contains(window, q) {
					hasSafetyQualifier = true
					break
				}
			}

			if !hasSafetyQualifier {
				// Replace the dangerous directive with a safe framing
				original := text[match[0]:match[1]]
				safe := "**Discuss with your doctor:** " + original
				text = text[:match[0]] + safe + text[match[1]:]
				// Recalculate lower after modification
				lower = strings.ToLower(text)
				break // re-scan from the top since indices shifted
			}
		}
	}

	return text
}

// appendDisclaimerIfNeeded adds a safety disclaimer to responses that contain
// medical recommendations but lack any disclaimer.
var medicalActionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(recommend|advise|suggest)\s+(you|that you|starting|taking|stopping)`),
	regexp.MustCompile(`(?i)\byou\s+(need|require|must)\s+(a |an |to )`),
	regexp.MustCompile(`(?i)\b(diagnosis|diagnose[sd]?)\b.*\byou\b`),
}

var disclaimerPatterns = []string{
	"not medical advice",
	"not a replacement",
	"not a substitute",
	"consult your",
	"talk to your doctor",
	"discuss with your",
	"speak with your",
	"ask your doctor",
	"healthcare provider",
	"healthcare professional",
	"medical professional",
}

const safetyDisclaimer = "\n\n---\n*This is informational only — not medical advice. Always discuss changes with your doctor.*"

func appendDisclaimerIfNeeded(text string) string {
	lower := strings.ToLower(text)

	// Check if the response contains medical action language
	hasAction := false
	for _, re := range medicalActionPatterns {
		if re.MatchString(lower) {
			hasAction = true
			break
		}
	}
	if !hasAction {
		return text
	}

	// Check if a disclaimer already exists
	for _, d := range disclaimerPatterns {
		if strings.Contains(lower, d) {
			return text
		}
	}

	return text + safetyDisclaimer
}
