package types

type PrimeArray struct {
	PrimeNumber []int
}

func NewPrimeArray() *PrimeArray {
	return &PrimeArray{
		PrimeNumber: []int{2781629, 2781637, 2781677, 2781683, 2781703, 2781707, 2781731},
	}
}
