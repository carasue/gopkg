package grace

import "log"

type logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

// stdLogger implements the logger interface.
type stdLogger struct{}

func (l *stdLogger) Infof(format string, args ...interface{}) {
	log.Printf("[INFO] "+format, args...)
}

func (l *stdLogger) Errorf(format string, args ...interface{}) {
	log.Printf("[ERROR] "+format, args...)
}

func (l *stdLogger) Fatalf(format string, args ...interface{}) {
	log.Fatalf("[FATAL] "+format, args...)
}
