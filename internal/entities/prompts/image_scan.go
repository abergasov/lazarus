package prompts

const ParseDocumentSystemPrompt = `Extract structured medical data from this document. Return ONLY a JSON object, no markdown fences, no explanation.

Schema:
{
  "lab_results": [{"name":"","value":0,"unit":"","range":"","flag":"normal|high|low","date":"YYYY-MM-DD"}],
  "medications": [{"name":"","dose":"","frequency":""}],
  "diagnoses": [{"code":"ICD-10","name":""}],
  "date": "YYYY-MM-DD",
  "category": "lab_result|specialist_visit|prescription|imaging|discharge|referral|vaccination|other",
  "specialty": "e.g. gastroenterology, urology, psychiatry, general_practice, endocrinology",
  "summary": "One-line summary of what this document is about"
}

Category guide:
- lab_result: blood work, urine analysis, any lab test results
- specialist_visit: doctor visit notes, examination results, consultation reports
- prescription: medication prescriptions
- imaging: X-ray, MRI, CT, ultrasound reports
- discharge: hospital discharge summaries
- referral: referral letters
- vaccination: vaccination records

Omit empty arrays. Always include date, category, specialty (if applicable), and summary.`
