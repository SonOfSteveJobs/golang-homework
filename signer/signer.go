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
	var wg1 sync.WaitGroup
	var wg2 sync.WaitGroup

	crc32results := make([]string, 0)
	md5results := make([]string, 0)

	for result := range in {
		md5Hash := DataSignerMd5(convertToString(result))

		wg1.Add(1)
		go func(result interface{}) {
			defer wg1.Done()
			hash := DataSignerCrc32(convertToString(result))
			crc32results = append(crc32results, hash)
		}(result)

		wg2.Add(1)
		go func(result interface{}) {
			defer wg2.Done()
			hash := DataSignerCrc32(md5Hash)
			md5results = append(md5results, hash)
		}(result)
	}
	wg1.Wait()
	wg2.Wait()

	for i := 0; i < len(md5results); i++ {
		out <- crc32results[i] + "~" + md5results[i]
	}
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
