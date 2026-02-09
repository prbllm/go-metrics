package threading

// Resetter определяет интерфейс для работы с функцией сброса произвольных структур.
type Resetter interface {

	// Reset метод сброса
	Reset()
}
