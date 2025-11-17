# Быстрые действия для улучшения тестового покрытия

**Приоритетные шаги, которые можно начать делать прямо сейчас**

---

## 🚀 Немедленные действия (сегодня)

### 1. Исправить проблему "store does not exist"

**Файл:** `tests/integration_test.go`, `tests/security_test.go`, `tests/end_to_end_test.go`

**Проблема:** Store keys не правильно монтируются в CommitMultiStore

**Решение:**
```go
// Создать helper функцию в tests/test_helpers.go
func NewTestContext() (sdk.Context, *store.CommitMultiStore) {
    db := dbm.NewMemDB()
    cms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
    
    // Создать все store keys
    identStoreKey := storetypes.NewKVStoreKey("ident")
    lizenzStoreKey := storetypes.NewKVStoreKey("lizenz")
    anteilStoreKey := storetypes.NewKVStoreKey("anteil")
    consensusStoreKey := storetypes.NewKVStoreKey("consensus")
    tKey := storetypes.NewTransientStoreKey("transient")
    
    // Смонтировать все stores
    cms.MountStoreWithDB(identStoreKey, storetypes.StoreTypeIAVL, db)
    cms.MountStoreWithDB(lizenzStoreKey, storetypes.StoreTypeIAVL, db)
    cms.MountStoreWithDB(anteilStoreKey, storetypes.StoreTypeIAVL, db)
    cms.MountStoreWithDB(consensusStoreKey, storetypes.StoreTypeIAVL, db)
    cms.MountStoreWithDB(tKey, storetypes.StoreTypeTransient, db)
    
    // Загрузить версию
    err := cms.LoadLatestVersion()
    if err != nil {
        panic(err)
    }
    
    // Создать контекст
    ctx := sdk.NewContext(cms, cmtproto.Header{}, false, log.NewNopLogger())
    
    return ctx, cms
}
```

**Действие:** Создать `tests/test_helpers.go` и использовать в всех test suite

---

### 2. Исправить "Account limit exceeded"

**Файл:** `tests/end_to_end_test.go`, `tests/integration_test.go`

**Проблема:** Лимиты слишком низкие для тестов

**Решение:**
```go
func (suite *EndToEndTestSuite) SetupTest() {
    // ...
    identParams := identtypes.DefaultParams()
    identParams.MaxIdentitiesPerAddress = 10000 // Увеличить для тестов
    suite.identKeeper.SetParams(suite.ctx, identParams)
    // ...
}
```

**Действие:** Увеличить лимиты в SetupTest всех test suite

---

### 3. Запустить и проанализировать падающие тесты

**Команды:**
```bash
# Запустить все тесты и сохранить вывод
go test ./... -v 2>&1 | tee test_output.log

# Запустить только падающие тесты
go test ./x/ident/keeper/... -v -run TestMsgServer
go test ./x/anteil/keeper/... -v -run TestMsgServer

# Анализировать ошибки
grep -i "error\|fail\|panic" test_output.log
```

**Действие:** Создать список всех ошибок и их причин

---

## 🔧 Быстрые исправления (эта неделя)

### 4. Исправить msg_server тесты

**Файлы:**
- `x/ident/keeper/msg_server_test.go`
- `x/anteil/keeper/msg_server_test.go`

**Типичные проблемы:**
1. Keeper не правильно инициализирован
2. Отсутствуют зависимости между keeper'ами
3. Неправильная валидация входных данных

**Решение:**
```go
// Убедиться, что все keeper'ы правильно связаны
func (suite *MsgServerTestSuite) SetupTest() {
    // Создать все keeper'ы
    suite.identKeeper = identkeeper.NewKeeper(...)
    suite.lizenzKeeper = lizenzkeeper.NewKeeper(...)
    suite.anteilKeeper = anteilkeeper.NewKeeper(...)
    
    // Установить зависимости
    suite.anteilKeeper.SetIdentKeeper(suite.identKeeper)
    suite.anteilKeeper.SetLizenzKeeper(suite.lizenzKeeper)
    
    // Установить параметры
    suite.identKeeper.SetParams(suite.ctx, identtypes.DefaultParams())
    // ...
}
```

**Действие:** Проверить и исправить SetupTest в msg_server тестах

---

### 5. Добавить недостающие тесты для types

**Файлы:**
- `x/ident/types/types_test.go` - добавить edge cases
- `x/anteil/types/types_test.go` - добавить edge cases
- `x/consensus/types/types_test.go` - расширить тесты

**Примеры:**
```go
// x/ident/types/types_test.go
func TestNewVerifiedAccount_EdgeCases(t *testing.T) {
    // Пустой адрес
    // Максимальная длина адреса
    // Специальные символы
    // Unicode символы
}

func TestValidateAccount_InvalidData(t *testing.T) {
    // Невалидные роли
    // Пустые хеши
    // Невалидные даты
}
```

**Действие:** Добавить 5-10 edge case тестов для каждого types модуля

---

### 6. Исправить benchmark тесты

**Файл:** `tests/benchmark_test.go`

**Проблема:** Проблемы с multi-store контекстом

**Решение:**
```go
func BenchmarkCreateOrder(b *testing.B) {
    // Создать отдельный контекст для каждого keeper
    storeKey := storetypes.NewKVStoreKey("test_anteil")
    tKey := storetypes.NewTransientStoreKey("test_transient")
    ctx := testutil.DefaultContext(storeKey, tKey)
    
    // Создать keeper
    keeper := anteilkeeper.NewKeeper(cdc, storeKey, paramStore)
    keeper.SetParams(ctx, anteiltypes.DefaultParams())
    
    b.ResetTimer()
    // ...
}
```

**Действие:** Исправить все benchmark тесты, использовать отдельные контексты

---

## 📈 Среднесрочные улучшения (2-4 недели)

### 7. Добавить тесты для app/ модуля

**Создать:** `app/app_test.go`

**Минимальный набор:**
```go
func TestNewApp(t *testing.T) {
    app := NewApp(...)
    require.NotNil(t, app)
}

func TestAppInitGenesis(t *testing.T) {
    // Тест инициализации genesis
}

func TestAppBeginBlocker(t *testing.T) {
    // Тест BeginBlocker
}
```

**Действие:** Создать базовые тесты для app модуля

---

### 8. Добавить тесты для economic engine

**Создать:** `x/anteil/keeper/economic_engine_test.go`

**Критические функции:**
```go
func TestCalculateOrderPrice(t *testing.T) {
    // Тест расчета цены
}

func TestMatchOrders(t *testing.T) {
    // Тест matching ордеров
}

func TestCalculateTradeFee(t *testing.T) {
    // Тест расчета комиссии
}
```

**Действие:** Добавить тесты для всех функций economic engine

---

### 9. Улучшить покрытие ZKP верификации

**Файл:** `x/ident/keeper/zkp_verifier_test.go`

**Добавить:**
```go
func TestVerifyZKPProof_ValidProof(t *testing.T) {
    // Валидное доказательство
}

func TestVerifyZKPProof_InvalidProof(t *testing.T) {
    // Невалидное доказательство
}

func TestVerifyZKPProof_ExpiredChallenge(t *testing.T) {
    // Истекший challenge
}
```

**Действие:** Добавить тесты для всех ZKP функций

---

## 🎯 Метрики для отслеживания

### Еженедельно проверять:
```bash
# Общее покрытие
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total

# Покрытие по модулям
go test ./x/ident/keeper/... -cover
go test ./x/lizenz/keeper/... -cover
go test ./x/anteil/keeper/... -cover
go test ./x/consensus/keeper/... -cover

# Количество падающих тестов
go test ./... 2>&1 | grep -c "FAIL"
```

### Цели на неделю:
- ✅ 0 падающих тестов
- ✅ Покрытие keeper'ов >60%
- ✅ Все integration тесты проходят

---

## 📋 Чеклист на сегодня

- [ ] Создать `tests/test_helpers.go` с функцией `NewTestContext`
- [ ] Использовать helper во всех test suite
- [ ] Увеличить лимиты в SetupTest
- [ ] Запустить все тесты и сохранить вывод
- [ ] Создать список всех ошибок
- [ ] Исправить хотя бы 1 падающий тест

---

## 📋 Чеклист на эту неделю

- [ ] Исправить все падающие msg_server тесты (7 тестов)
- [ ] Исправить проблему "store does not exist" (13 тестов)
- [ ] Исправить проблему "Account limit exceeded"
- [ ] Исправить benchmark тесты
- [ ] Добавить 10+ edge case тестов для types
- [ ] Покрытие keeper'ов >60%

---

## 🆘 Если застряли

### Проблема: Не понимаю, почему тест падает
**Решение:**
```bash
# Запустить с максимальным выводом
go test -v -run TestName 2>&1 | tee debug.log

# Использовать дебаггер
dlv test -- -test.run TestName
```

### Проблема: Тесты слишком медленные
**Решение:**
```go
// Использовать -short флаг
if testing.Short() {
    t.Skip("Skipping slow test")
}

// Параллельное выполнение
t.Parallel()
```

### Проблема: Не знаю, что тестировать
**Решение:**
1. Посмотреть на функции без тестов: `go test -coverprofile=coverage.out && go tool cover -func=coverage.out | grep "0.0%"`
2. Начать с критических функций (keeper методы)
3. Добавить edge cases для существующих тестов

---

**Начните с первых 3 пунктов "Немедленные действия" - они дадут быстрый результат!**

