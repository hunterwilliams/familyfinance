package money

// Satang is the smallest THB unit (1 THB = 100 satang).
type Satang int64

func THB(amount float64) Satang {
	return Satang(amount * 100)
}

func (s Satang) THB() float64 {
	return float64(s) / 100
}
