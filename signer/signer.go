package main

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

// сюда писать код
// сохранил типизацию сигнатур функций как в тестах. Если бы я писал сам - типизировал бы каналы и добавил возвращаемые значения.
func main() {
	inputData := []int{0, 1, 1, 2, 3, 5, 8}

	hashSignJobs := []job{
		job(func(in, out chan interface{}) {
			for _, fibNum := range inputData {
				out <- fibNum
			}
		}),
		job(SingleHash),
		job(MultiHash),
		job(CombineResults),
	}

	start := time.Now()

	ExecutePipeline(hashSignJobs...)

	end := time.Since(start)
	fmt.Printf("Execution time: %s\n", end)
}

func ExecutePipeline(jobs ...job) {
	var wg sync.WaitGroup
	wg.Add(len(jobs))

	in := make(chan interface{})

	for _, j := range jobs {
		out := make(chan interface{})

		go func(job job, in, out chan interface{}) {
			defer wg.Done()
			defer close(out)
			job(in, out)
		}(j, in, out)

		in = out
	}
	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	var wg sync.WaitGroup

	for result := range in {
		wg.Add(1)
		go func(result interface{}) {
			defer wg.Done()
			hash := DataSignerCrc32(convertToString(result))
			out <- hash
			fmt.Printf("SingleHash: %v\n", hash)
		}(result)
	}
	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	for result := range in {
		out <- result
		fmt.Printf("MultiHash: %v\n", result)
	}
}

func CombineResults(in, out chan interface{}) {
	for result := range in {
		fmt.Printf("CombineResults: %v\n", result)
	}
}

func convertToString(value interface{}) string {
	switch v := value.(type) {
	case int:
		return strconv.Itoa(v)
	case string:
		return v
	default:
		//панику кидаю потому что наша программа вообще не предусматривает ошибки
		panic("unknown type")
	}
}
