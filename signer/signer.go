package main

import (
	"fmt"
	"sync"
	"time"
)

// сюда писать код
// сохранил типизацию сигнатур функций как в тестах. Если бы я писал сам - типизировал бы каналы и добавил возвращаемые значения (вроде как каждая из функций может упасть с ошибкой).
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
	for result := range in {
		out <- result
		fmt.Printf("SingleHash: %d\n", result)
	}
}

func MultiHash(in, out chan interface{}) {
	for result := range in {
		out <- result
		fmt.Printf("MultiHash: %d\n", result)
	}
}

func CombineResults(in, out chan interface{}) {
	for result := range in {
		fmt.Printf("CombineResults: %d\n", result)
	}
}
