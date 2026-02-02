package main

import (
	"sort"
	"strconv"
	"strings"
	"sync"
)

// сюда писать код
// сохранил типизацию сигнатур функций как в тестах. Если бы я писал сам - типизировал бы каналы и добавил возвращаемые значения.

// для теста:
// func main() {
// 	inputData := []int{0, 1, 1, 2, 3, 5, 8}

// 	hashSignJobs := []job{
// 		job(func(in, out chan interface{}) {
// 			for _, fibNum := range inputData {
// 				out <- fibNum
// 			}
// 		}),
// 		job(SingleHash),
// 		job(MultiHash),
// 		job(CombineResults),
// 	}

// 	start := time.Now()

// 	ExecutePipeline(hashSignJobs...)

// 	end := time.Since(start)
// 	fmt.Printf("Execution time: %s\n", end)
// }

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
		md5Hash := DataSignerMd5(convertToString(result))

		wg.Add(1)
		go func(result interface{}) {
			defer wg.Done()
			var hashCrc32 string
			var hashMd5 string
			var wgInner sync.WaitGroup

			wgInner.Add(2)
			go func() {
				defer wgInner.Done()
				hashCrc32 = DataSignerCrc32(convertToString(result))
			}()
			go func() {
				defer wgInner.Done()
				hashMd5 = DataSignerCrc32(md5Hash)
			}()
			wgInner.Wait()

			out <- hashCrc32 + "~" + hashMd5
		}(result)
	}
	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	const iterationsPerElement = 6
	var wg sync.WaitGroup

	for result := range in {
		wg.Add(1)
		go func(result interface{}) {
			defer wg.Done()
			var wgInner sync.WaitGroup
			hashes := make([]string, iterationsPerElement)

			wgInner.Add(iterationsPerElement)
			for th := 0; th < iterationsPerElement; th++ {
				go func(th int, result interface{}) {
					defer wgInner.Done()
					hashes[th] = DataSignerCrc32(strconv.Itoa(th) + convertToString(result))
				}(th, result)
			}

			wgInner.Wait()
			string := strings.Join(hashes, "")
			out <- string
		}(result)
	}

	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	results := make([]string, 0)

	for result := range in {
		results = append(results, convertToString(result))
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i] < results[j]
	})

	resultString := strings.Join(results, "_")
	out <- resultString
}

func convertToString(value interface{}) string {
	switch v := value.(type) {
	case int:
		return strconv.Itoa(v)
	case string:
		return v
	default:
		//панику кидаю потому что наша программа вообще не предусматривает ошибки (?)
		panic("unknown type")
	}
}
