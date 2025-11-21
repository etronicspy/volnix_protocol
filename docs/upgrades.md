# 🔄 Upgrade механизм

## Обзор

Volnix Protocol включает механизм обновлений сети, позволяющий выполнять миграции состояния и обновления без остановки сети. Upgrade механизм поддерживает:

- Миграции состояния модулей
- Обновления параметров
- Изменения структуры данных
- Версионирование приложения

## Архитектура

### Upgrade Manager

`UpgradeManager` управляет всеми upgrade handlers и миграциями:

```go
type UpgradeManager struct {
    handlers map[string]UpgradeHandler
    logger   sdklog.Logger
}
```

### Upgrade Handler

Upgrade handler - это функция, которая выполняет миграцию для конкретной версии:

```go
type UpgradeHandler func(ctx sdk.Context, plan UpgradePlan, app *VolnixApp) error
```

### Upgrade Plan

Upgrade plan определяет, когда и как выполнить обновление:

```go
type UpgradePlan struct {
    Name   string  // Версия обновления (например, "v0.2.0")
    Height int64   // Высота блока для выполнения обновления
    Info   string  // Описание обновления
}
```

## Регистрация Upgrade Handlers

Upgrade handlers регистрируются в `SetupUpgradeHandlers`:

```go
func SetupUpgradeHandlers(um *UpgradeManager, app *VolnixApp) {
    // Регистрация handler для v0.2.0
    um.RegisterUpgradeHandler("v0.2.0", func(ctx sdk.Context, plan UpgradePlan, app *VolnixApp) error {
        return MigrateToV0_2_0(ctx, app)
    })
}
```

## Создание миграций

### Пример миграции модуля

```go
func MigrateToV0_2_0(ctx sdk.Context, app *VolnixApp) error {
    // Миграция ident модуля
    if err := migrateIdentModuleV0_2_0(ctx, app); err != nil {
        return fmt.Errorf("ident module migration failed: %w", err)
    }
    
    // Миграция lizenz модуля
    if err := migrateLizenzModuleV0_2_0(ctx, app); err != nil {
        return fmt.Errorf("lizenz module migration failed: %w", err)
    }
    
    return nil
}
```

### Миграция конкретного модуля

```go
func migrateIdentModuleV0_2_0(ctx sdk.Context, app *VolnixApp) error {
    // Получить все аккаунты
    accounts, err := app.identKeeper.GetAllVerifiedAccounts(ctx)
    if err != nil {
        return fmt.Errorf("failed to get verified accounts: %w", err)
    }
    
    // Выполнить миграцию для каждого аккаунта
    for _, account := range accounts {
        // Добавить новые поля, обновить структуру и т.д.
        // account.NewField = defaultValue
        
        // Сохранить обновленный аккаунт
        if err := app.identKeeper.UpdateVerifiedAccount(ctx, account); err != nil {
            return fmt.Errorf("failed to update account: %w", err)
        }
    }
    
    return nil
}
```

## Выполнение обновлений

### Автоматическое выполнение

Upgrade manager автоматически проверяет необходимость обновления в начале каждого блока:

```go
func (um *UpgradeManager) CheckUpgradeNeeded(ctx sdk.Context, app *VolnixApp) error {
    currentHeight := ctx.BlockHeight()
    
    // Проверить, нужен ли upgrade на текущей высоте
    for _, plan := range upgradePlans {
        if currentHeight == plan.Height {
            return um.ExecuteUpgrade(ctx, plan, app)
        }
    }
    
    return nil
}
```

### Ручное выполнение

Можно выполнить upgrade вручную через governance proposal или CLI команду:

```go
plan := UpgradePlan{
    Name:   "v0.2.0",
    Height: ctx.BlockHeight(),
    Info:   "Migration to v0.2.0",
}

if err := upgradeManager.ExecuteUpgrade(ctx, plan, app); err != nil {
    return err
}
```

## Best Practices

### Идемпотентность

Миграции должны быть идемпотентными - их можно безопасно выполнять несколько раз:

```go
func migrateIdentModuleV0_2_0(ctx sdk.Context, app *VolnixApp) error {
    accounts, err := app.identKeeper.GetAllVerifiedAccounts(ctx)
    if err != nil {
        return err
    }
    
    for _, account := range accounts {
        // Проверить, была ли миграция уже выполнена
        if account.NewField != "" {
            continue // Уже мигрировано
        }
        
        // Выполнить миграцию
        account.NewField = defaultValue
        if err := app.identKeeper.UpdateVerifiedAccount(ctx, account); err != nil {
            return err
        }
    }
    
    return nil
}
```

### Обработка ошибок

Всегда обрабатывайте ошибки и предоставляйте понятные сообщения:

```go
func MigrateToV0_2_0(ctx sdk.Context, app *VolnixApp) error {
    if err := migrateIdentModuleV0_2_0(ctx, app); err != nil {
        return fmt.Errorf("ident module migration failed: %w", err)
    }
    
    // Если одна миграция не удалась, остальные не выполняются
    // Это обеспечивает целостность данных
}
```

### Тестирование

Всегда тестируйте миграции на testnet перед mainnet:

1. Создайте тестовую сеть с данными, похожими на production
2. Выполните миграцию на testnet
3. Проверьте целостность данных после миграции
4. Убедитесь, что все модули работают корректно

### Документация

Документируйте все изменения в миграциях:

```go
// MigrateToV0_2_0 performs state migration to version 0.2.0
// Changes:
// - Added NewField to VerifiedAccount
// - Updated Lizenz structure to include metadata
// - Migrated consensus parameters to new format
func MigrateToV0_2_0(ctx sdk.Context, app *VolnixApp) error {
    // ...
}
```

## Интеграция с Governance

Для production использования рекомендуется интегрировать upgrade механизм с governance модулем:

1. Создать governance proposal для upgrade
2. Получить одобрение от валидаторов
3. Выполнить upgrade на указанной высоте блока

Пример:

```go
// В governance модуле
func (k Keeper) SubmitUpgradeProposal(ctx sdk.Context, plan UpgradePlan) error {
    // Создать proposal
    // После одобрения, upgrade будет выполнен на plan.Height
}
```

## Откат (Rollback)

В случае проблем с upgrade, можно выполнить откат:

1. Остановить узел
2. Восстановить состояние из бэкапа
3. Откатить код к предыдущей версии
4. Перезапустить узел

**Важно**: Всегда создавайте бэкапы перед выполнением upgrade!

## Примеры использования

### Добавление нового поля в аккаунт

```go
func migrateIdentModuleV0_2_0(ctx sdk.Context, app *VolnixApp) error {
    accounts, err := app.identKeeper.GetAllVerifiedAccounts(ctx)
    if err != nil {
        return err
    }
    
    for _, account := range accounts {
        // Добавить новое поле с дефолтным значением
        if account.Metadata == nil {
            account.Metadata = &identv1.AccountMetadata{
                CreatedAt: timestamppb.Now(),
            }
        }
        
        if err := app.identKeeper.UpdateVerifiedAccount(ctx, account); err != nil {
            return err
        }
    }
    
    return nil
}
```

### Изменение структуры параметров

```go
func migrateLizenzModuleV0_2_0(ctx sdk.Context, app *VolnixApp) error {
    // Получить текущие параметры
    params := app.lizenzKeeper.GetParams(ctx)
    
    // Обновить параметры
    params.NewParameter = defaultValue
    
    // Сохранить обновленные параметры
    app.lizenzKeeper.SetParams(ctx, params)
    
    return nil
}
```

## Troubleshooting

### Upgrade не выполняется

1. Проверьте, что upgrade handler зарегистрирован
2. Убедитесь, что высота блока совпадает с plan.Height
3. Проверьте логи на наличие ошибок

### Ошибки миграции

1. Проверьте логи для детальной информации об ошибке
2. Убедитесь, что все зависимости миграции выполнены
3. Проверьте целостность данных перед миграцией

### Частичная миграция

Если миграция была прервана:
1. Проверьте, какие данные были мигрированы
2. Выполните миграцию повторно (она должна быть идемпотентной)
3. При необходимости выполните откат

## Дополнительные ресурсы

- [Cosmos SDK Upgrades](https://docs.cosmos.network/main/core/upgrade)
- [Migration Best Practices](https://docs.cosmos.network/main/building-modules/migrations)



