package retry

type ErrorClassification int

const (
	NonRetriable ErrorClassification = iota
	Retriable
)

type ErrorClassifier interface {
	Classify(error) ErrorClassification
}
