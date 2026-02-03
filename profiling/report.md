# Отчет по профилированию

## CPU:

1. Запустил тест с профилированием CPU:
   ```
   go test -bench=BenchmarkSlow -cpuprofile=cpu.out
   go tool pprof -http=:8083 cpu.out
   ```
2. Нашел в дереве SlowSearch:
     SlowSearch - 0 of 64 (24,33% - доля общего времени) 
3. Перешел во View -> Source -> поиск по "SlowSearch"
4. Функции с самым большим flat:
    ```
    json.Unmarshal([]byte(line), &user) - 20
    regexp.MatchString("Android", browser) - 24
    regexp.MatchString("MSIE", browser) - 16
    ```
5. Также подвечены строки:
    ```
    file, err := os.Open(filePath) - 2
    fileContents, err := ioutil.ReadAll(file) - 1
    lines := strings.Split(string(fileContents), "\n") - 1
    ```

## Memory:

1. Запустил тест с профилированием памяти:
   ```
   go test -bench=BenchmarkSlow -memprofile=mem.out
   go tool pprof -http=:8084 mem.out
   ```
2. Перешел во View -> Source -> поиск по "SlowSearch"
3. Вижу самые тяжелые операции:
    ```
    ioutil.ReadAll(file) - 289mb
    strings.Split(string(fileContents), "\n") - 54mb
    user := make(map[string]interface{}) - 4.5mb
    err := json.Unmarshal([]byte(line), &user) - 198mb
    users = append(users, user) - 2mb
    regexp.MatchString("Android", browser) - 783mb
    regexp.MatchString("MSIE", browser) - 507mb
    foundUsers += fmt.Sprintf("[%d] %s <%s>\n", i, user["name"], email) - 21.5mb
    ```


## Выводы:
1. regexp.MatchString - самые тяжелые операции. На каждой итерации создается новая регулярка. r := regexp.MustCompile("@") хотя бы один раз создается и даже не подсвеичвается профайлером. Тут от регулярок сто процентов можно отказаться, потому что для таких простых вещей можно использовать пакет strings.
2. ReadAll читает всё в память одним куском, но можно читать построчно
3. strings.Split(string(fileContents), "\n") - конвертация байтов  в строку + слайс с копиями каждой строки
4. json.Unmarshal - использовать структуру вместо мапы
5. fmt.Sprintf - можно использовать strings.Builder (сам копилятор говорит)
6. зачем собирать всех пользователей в слайс (один цикл), а потом проходиться по ним (второй цикл), если можно этого не делать

## Первая итерация:
- ns/op: 4,241,022  (нужно < 2,782,432)
- B/op: 2,122,494  (нужно < 559,910)
- allocs/op: 17,335  (нужно < 10,422)

ускорение по сравнению со slow в 3 раза, по памяти в 9 раз

1. Запустил бенч по CPU, вижу что основная проблема в json.Unmarshal([]byte(line), &currentUser) (140ms). Конвертирую строку в байты, не знал что есть scanner.Bytes(). Это дало буст:
```
    До    BenchmarkFast-12    277	   4.24М ns/op	   2.12М B/op	     17335 allocs/op
    После BenchmarkFast-12    333	   3.9М ns/op	    933К B/op	       15332 allocs/op
```
2. Далее бенчи по памяти и CPU показали что я упираюсь только в json.Unmarshal, поэтому просто заюзал easyjson (сгенерированый код лежит в папке user).

## Вторая итерация
- ns/op: 986,535  (в 3 раза лучше эталона)
- B/op: 572,642  (чуть больше эталона 559,910)
- allocs/op: 7332  (лучше эталона в 1,4 раза)
