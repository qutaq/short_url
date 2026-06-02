# go-musthave-shortener-tpl

Шаблон репозитория для трека «Сервис сокращения URL».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-shortener-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## Сравнение профилей памяти

Команда:

```
go tool pprof -top -diff_base="profiles\base.pprof" "profiles\result.pprof"
```

Результат:

```
File: service.test.exe
Type: alloc_space
Time: 2026-05-17 16:02:16 +04
Showing nodes accounting for 84.82MB, 2.47% of 3429.85MB total
Dropped 4 nodes (cum <= 17.15MB)
      flat  flat%   sum%        cum   cum%
  765.02MB 22.30% 22.30%   765.02MB 22.30%  github.com/qutaq/short_url/internal/service.(*Shortener).makeShortURL (inline)
 -652.08MB 19.01%  3.29%    42.79MB  1.25%  github.com/qutaq/short_url/internal/service.(*Shortener).GetUserURLs
  -35.52MB  1.04%  2.26%    17.54MB  0.51%  github.com/qutaq/short_url/internal/service.(*Shortener).ShortenBatch
  -29.50MB  0.86%  1.40%    26.02MB  0.76%  github.com/qutaq/short_url/internal/service.(*Shortener).Shorten
   23.52MB  0.69%  2.08%    23.52MB  0.69%  github.com/qutaq/short_url/internal/repository.(*MemoryRepository).Save
   21.56MB  0.63%  2.71%    21.56MB  0.63%  github.com/qutaq/short_url/internal/repository.(*MemoryRepository).SaveBatch
     -10MB  0.29%  2.42%      -10MB  0.29%  fmt.Sprintf
       6MB  0.17%  2.59%    13.54MB  0.39%  github.com/qutaq/short_url/internal/service.BenchmarkShortenerShortenBatch
   -5.16MB  0.15%  2.44%    -5.16MB  0.15%  github.com/qutaq/short_url/internal/repository.(*MemoryRepository).GetURLsByUser
       4MB  0.12%  2.56%    29.52MB  0.86%  github.com/qutaq/short_url/internal/service.BenchmarkShortenerShorten
   -1.50MB 0.044%  2.52%    -1.50MB 0.044%  github.com/qutaq/short_url/internal/service.(*Shortener).generateUniqueID
   -0.52MB 0.015%  2.50%    -0.52MB 0.015%  regexp.(*bitState).reset
   -0.50MB 0.015%  2.49%    -0.50MB 0.015%  runtime.allocm
   -0.50MB 0.015%  2.47%    -0.50MB 0.015%  testing.(*B).ResetTimer
         0     0%  2.47%    42.79MB  1.25%  github.com/qutaq/short_url/internal/service.BenchmarkShortenerGetUserURLs
         0     0%  2.47%    -0.52MB 0.015%  regexp.(*Regexp).MatchString (inline)
         0     0%  2.47%    -0.52MB 0.015%  regexp.(*Regexp).backtrack
         0     0%  2.47%    -0.52MB 0.015%  regexp.(*Regexp).doExecute
         0     0%  2.47%    -0.52MB 0.015%  regexp.(*Regexp).doMatch (inline)
         0     0%  2.47%    -0.50MB 0.015%  runtime.mcall
         0     0%  2.47%    -0.50MB 0.015%  runtime.newm
         0     0%  2.47%    -0.50MB 0.015%  runtime.park_m
         0     0%  2.47%    -0.50MB 0.015%  runtime.resetspinning
         0     0%  2.47%    -0.50MB 0.015%  runtime.schedule
         0     0%  2.47%    -0.50MB 0.015%  runtime.startm
         0     0%  2.47%    -0.50MB 0.015%  runtime.wakep
         0     0%  2.47%    85.34MB  2.49%  testing.(*B).run1.func1
         0     0%  2.47%    85.34MB  2.49%  testing.(*B).runN
         0     0%  2.47%    -0.52MB 0.015%  testing.(*matcher).fullName
         0     0%  2.47%     0.50MB 0.015%  testing.benchmarkName
         0     0%  2.47%    -0.52MB 0.015%  testing.simpleMatch.matches
         0     0%  2.47%    -0.52MB 0.015%  testing/internal/testdeps.TestDeps.MatchString
```
