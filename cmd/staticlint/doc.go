// Package main contains staticlint multichecker for the short_url project.
//
// # Запуск multichecker
//
// Локальный запуск по всему проекту:
//
//	go run ./cmd/staticlint ./...
//
// Запуск по конкретному пакету:
//
//	go run ./cmd/staticlint ./internal/service
//
// Сборка бинаря:
//
//	go build -o staticlint ./cmd/staticlint
//	./staticlint ./...
//
// # Состав анализаторов
//
// staticlint объединяет четыре группы проверок.
//
//  1. Стандартные анализаторы из
//     golang.org/x/tools/go/analysis/passes:
//
//     - appends: лишние append-операции;
//     - asmdecl: соответствие объявлений и asm;
//     - assign: бесполезные присваивания;
//     - atomic: типичные ошибки sync/atomic;
//     - bools: логические выражения с ошибками;
//     - buildtag: корректность build tags;
//     - cgocall: корректность cgo-вызовов;
//     - composite: неименованные поля struct literals;
//     - copylock: копирование значений с lock-семантикой;
//     - defers: частые ошибки defer в циклах;
//     - directive: корректность compiler directives;
//     - errorsas: корректность errors.As;
//     - framepointer: проверки frame pointer;
//     - httpmux: конфликтующие HTTP-маршруты;
//     - ifaceassert: невозможные type assertions;
//     - loopclosure: захват переменных цикла;
//     - lostcancel: потерянный cancel context;
//     - nilfunc: сравнение функций с nil;
//     - nilness: анализ недостижимых nil-путей;
//     - pkgfact: инфраструктурный анализ пакетных фактов;
//     - printf: корректность printf-подобных вызовов;
//     - reflectvaluecompare: сравнение reflect.Value;
//     - shift: некорректные сдвиги;
//     - sigchanyzer: ошибки работы с signal channels;
//     - slog: корректность structured logging в log/slog;
//     - sortslice: ошибки сортировки слайсов;
//     - stdmethods: сигнатуры стандартных интерфейсных методов;
//     - stdversion: использование API новее целевой Go-версии;
//     - stringintconv: подозрительные string(int) преобразования;
//     - structtag: корректность struct tags;
//     - testinggoroutine: ошибки в тестах с goroutine;
//     - tests: типичные проблемы в *_test.go;
//     - timeformat: неверные time format layout-строки;
//     - unmarshal: некорректные аргументы unmarshal;
//     - unreachable: недостижимый код;
//     - unsafeptr: небезопасные конверсии unsafe.Pointer;
//     - unusedresult: игнорирование важных результатов;
//     - waitgroup: типичные ошибки sync.WaitGroup.
//
//  2. Все SA-анализаторы staticcheck (пакет honnef.co/go/tools/staticcheck):
//     проверяют correctness, reliability и performance issues (класс SA*).
//
//  3. Анализаторы других классов staticcheck:
//
//     - весь набор quickfix (класс QF*) для автоматизируемых рефакторингов и
//     улучшений читаемости.
//
//  4. Дополнительные публичные анализаторы:
//
//     - bodyclose (github.com/timakin/bodyclose): гарантирует закрытие HTTP response body;
//     - exhaustive (github.com/nishanths/exhaustive): проверяет исчерпывающую
//     обработку enum-подобных типов в switch;
//     - noosexit (локальный, cmd/staticlint/noosexit): запрещает прямой вызов os.Exit
//     внутри main.main в пакете main.
//
// # Кастомное правило noosexit
//
// Анализатор noosexit обходит AST файлов пакета main, находит функцию main и
// репортит вызовы os.Exit(...), если идентификатор os действительно резолвится
// к стандартному пакету os через types.Info. Рекомендуемая замена: возврат
// ошибки из вспомогательной run()-функции и обработка через логирование.
package main
