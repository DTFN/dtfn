package types

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPrime(t *testing.T) {
	require.Equal(t, true, isPrimeOptimize(5))
	require.Equal(t, false, isPrimeOptimize(99))

	isPrime := true
	primeArray := NewPrimeArray()
	for i := 0; i < len(primeArray.PrimeNumber); i++ {
		isPrime = isPrime && isPrimeOptimize(primeArray.PrimeNumber[i])
	}
	require.Equal(t, true, isPrime)

	primeArray.PrimeNumber = append(primeArray.PrimeNumber,99)
	for i := 0; i < len(primeArray.PrimeNumber); i++ {
		isPrime = isPrime && isPrimeOptimize(primeArray.PrimeNumber[i])
	}
	require.Equal(t, false, isPrime)
}

func isPrimeOptimize(n int) bool {
	i := 2
	//这里的(n / i）是从筛选法中获取的比例值
	// i/j 米勒/拉宾检验
	for ; i <= (n / i); i++ {
		if n%i == 0 {
			break
		}
	}

	if i > (n / i) {
		//NSLog(@"此数是素数 %d", n);
		return true
	}

	return false
}
