# Диаграммы экономической модели Volnix Protocol

## Схема взаимодействия токенов

```mermaid
graph TB
    subgraph "Трехтокенная система"
        WRT[WRT - Ценность и Голос<br/>21M максимум<br/>Халвинг каждые 210K блоков]
        LZN[LZN - Лицензия на майнинг<br/>1M фиксированная эмиссия<br/>Торгуемый токен]
        ANT[ANT - Право на производительность<br/>Внутренняя единица учета<br/>Неторгуемый]
    end

    subgraph "Роли участников"
        GUEST[Гость<br/>Неверифицированный]
        CITIZEN[Гражданин<br/>Верифицированный]
        VALIDATOR[Валидатор<br/>Оператор узла]
    end

    subgraph "Экономические потоки"
        EMISSION[Базовая эмиссия WRT]
        MARKET[Внутренний рынок ANT]
        AUCTION[Аукционы блоков]
        FEES[Комиссии за транзакции]
    end

    %% Связи токенов и ролей
    GUEST -->|Может держать| WRT
    GUEST -->|Может держать| LZN
    GUEST -->|Верификация| CITIZEN
    GUEST -->|Верификация| VALIDATOR

    CITIZEN -->|Получает| ANT
    CITIZEN -->|Продает на| MARKET
    CITIZEN -->|Голосует с| WRT

    VALIDATOR -->|Активирует| LZN
    VALIDATOR -->|Покупает на| MARKET
    VALIDATOR -->|Участвует в| AUCTION
    VALIDATOR -->|Получает| EMISSION
    VALIDATOR -->|Получает| FEES

    %% Экономические потоки
    EMISSION -->|Распределяется по| LZN
    MARKET -->|Торгует| ANT
    AUCTION -->|Использует| ANT
    AUCTION -->|Генерирует| FEES

    style WRT fill:#FFD700
    style LZN fill:#87CEEB
    style ANT fill:#98FB98
    style CITIZEN fill:#FFA07A
    style VALIDATOR fill:#DDA0DD
```

## Двухконтурная экономическая модель

```mermaid
graph LR
    subgraph "Контур 1: Безопасность и Стабильность"
        V1[Валидаторы]
        LZN_POOL[Пул активированных LZN]
        BASE_REWARD[Базовая награда WRT]
        
        V1 -->|Активируют| LZN_POOL
        LZN_POOL -->|Доля от| BASE_REWARD
        BASE_REWARD -->|Пассивный доход| V1
    end

    subgraph "Контур 2: Производительность и Скорость"
        C1[Граждане]
        V2[Валидаторы]
        ANT_MARKET[Внутренний рынок ANT]
        BLOCK_AUCTION[Аукционы блоков]
        TX_FEES[Комиссии за транзакции]
        
        C1 -->|Продают ANT| ANT_MARKET
        V2 -->|Покупают ANT| ANT_MARKET
        ANT_MARKET -->|Поставляет ANT для| BLOCK_AUCTION
        V2 -->|Участвуют в| BLOCK_AUCTION
        BLOCK_AUCTION -->|Генерирует| TX_FEES
        TX_FEES -->|Активный доход| V2
    end

    subgraph "Связующее звено: MOA"
        MOA[Минимальное Обязательство Активности]
        V1 -.->|Обязаны участвовать| MOA
        MOA -.->|Требует покупки ANT| ANT_MARKET
    end

    style V1 fill:#DDA0DD
    style V2 fill:#DDA0DD
    style C1 fill:#FFA07A
    style MOA fill:#FFB6C1
```

## Поток создания и использования ANT

```mermaid
sequenceDiagram
    participant C as Гражданин
    participant M as Внутренний рынок
    participant V as Валидатор
    participant A as Аукцион блока
    participant N as Сеть

    Note over C: Автоматическое начисление
    C->>C: +10 ANT в день
    
    Note over C,M: Продажа ANT
    C->>M: Создать ордер на продажу
    M->>M: Добавить в книгу ордеров
    
    Note over V,M: Покупка ANT (MOA)
    V->>M: Купить ANT для MOA
    M->>V: Передать ANT
    M->>C: Передать WRT
    
    Note over V,A: Участие в аукционе
    V->>A: Подать заявку с ANT
    A->>A: Провести слепой аукцион
    A->>V: Объявить победителя
    
    Note over V,N: Создание блока
    V->>N: Создать блок
    N->>V: Комиссии за транзакции
    A->>A: Сжечь использованные ANT
```

## Механизм халвинга и динамического времени блока

```mermaid
graph TD
    subgraph "Система халвинга"
        BLOCKS[Количество блоков]
        HALVING_CHECK{Блок кратен 210,000?}
        REWARD_CALC[Расчет награды]
        CURRENT_REWARD[Текущая награда за блок]
        
        BLOCKS --> HALVING_CHECK
        HALVING_CHECK -->|Да| REWARD_CALC
        HALVING_CHECK -->|Нет| CURRENT_REWARD
        REWARD_CALC --> CURRENT_REWARD
    end

    subgraph "Динамическое время блока"
        ANT_USAGE[Использование ANT]
        NETWORK_ACTIVITY[Активность сети]
        BLOCK_TIME_CALC[Расчет времени блока]
        BLOCK_TIME[Время блока: 1-60 сек]
        
        ANT_USAGE --> NETWORK_ACTIVITY
        NETWORK_ACTIVITY --> BLOCK_TIME_CALC
        BLOCK_TIME_CALC --> BLOCK_TIME
    end

    subgraph "Адаптивный халвинг"
        BLOCK_TIME --> HALVING_SCHEDULE[График халвинга]
        CURRENT_REWARD --> HALVING_SCHEDULE
        HALVING_SCHEDULE --> ECONOMIC_CYCLE[Экономический цикл]
    end

    style CURRENT_REWARD fill:#FFD700
    style BLOCK_TIME fill:#87CEEB
    style ECONOMIC_CYCLE fill:#98FB98
```

## Модель ценообразования на внутреннем рынке

```mermaid
graph TB
    subgraph "Факторы спроса на ANT"
        VALIDATORS[Количество валидаторов]
        MOA_REQ[Требования MOA]
        AUCTION_COMP[Конкуренция в аукционах]
        TX_VOLUME[Объем транзакций]
        
        VALIDATORS --> DEMAND[Спрос на ANT]
        MOA_REQ --> DEMAND
        AUCTION_COMP --> DEMAND
        TX_VOLUME --> DEMAND
    end

    subgraph "Факторы предложения ANT"
        CITIZENS[Количество граждан]
        ANT_ACCRUAL[Начисление ANT]
        HOLDING_STRATEGY[Стратегии накопления]
        
        CITIZENS --> SUPPLY[Предложение ANT]
        ANT_ACCRUAL --> SUPPLY
        HOLDING_STRATEGY --> SUPPLY
    end

    subgraph "Ценообразование"
        DEMAND --> PRICE_CALC[Расчет цены]
        SUPPLY --> PRICE_CALC
        PRICE_CALC --> ANT_PRICE[Цена ANT в WRT]
        
        ANT_PRICE --> MARKET_FEEDBACK[Обратная связь рынка]
        MARKET_FEEDBACK --> DEMAND
        MARKET_FEEDBACK --> SUPPLY
    end

    style ANT_PRICE fill:#FFD700
    style DEMAND fill:#FF6B6B
    style SUPPLY fill:#4ECDC4
```

## Экономические стимулы по ролям

```mermaid
mindmap
  root((Экономические стимулы))
    Гости
      Базовый функционал
        Хранение WRT/LZN
        Простые транзакции
      Стимулы к верификации
        Доступ к доходам
        Расширенные возможности
    Граждане
      Источники дохода
        Автоматическое начисление ANT
        Продажа ANT на рынке
        Участие в DAO
      Стратегии оптимизации
        Мониторинг цен
        Накопление при низких ценах
        Продажа при пиках спроса
    Валидаторы
      Пассивный доход
        Доля от базовой эмиссии
        Пропорционально LZN
      Активный доход
        Комиссии за транзакции
        Выигрыш аукционов блоков
      Расходы и риски
        Активация LZN
        Покупка ANT
        Операционные расходы
        Штрафы за нарушение MOA
```

## Система мониторинга MOA

```mermaid
stateDiagram-v2
    [*] --> Active: Валидатор активирует LZN
    
    Active --> Compliant: MOA >= 100%
    Active --> Warning: MOA 90-99%
    Active --> Penalty: MOA 70-89%
    Active --> Suspension: MOA 50-69%
    Active --> Deactivation: MOA < 50%
    
    Compliant --> Active: Новый период проверки
    Warning --> Active: Исправление активности
    Penalty --> Active: Исправление активности
    Suspension --> Active: Исправление активности
    
    Warning --> Penalty: Ухудшение показателей
    Penalty --> Suspension: Ухудшение показателей
    Suspension --> Deactivation: Ухудшение показателей
    
    Deactivation --> [*]: Принудительная деактивация LZN
    
    note right of Compliant
        Полная награда
        Нормальное участие
    end note
    
    note right of Warning
        Уведомление
        Полная награда
    end note
    
    note right of Penalty
        Снижение награды на 25%
    end note
    
    note right of Suspension
        Снижение награды на 50%
    end note
    
    note right of Deactivation
        Потеря статуса валидатора
        Разморозка LZN
    end note
```

## Защита от экономических атак

```mermaid
graph TB
    subgraph "Типы атак"
        SYBIL[Атака Сивиллы]
        CONCENTRATION[Концентрация власти]
        MARKET_MANIP[Манипуляции рынком]
        NOTHING_AT_STAKE[Nothing at Stake]
    end

    subgraph "Механизмы защиты"
        ZKP[ZKP верификация]
        LZN_LIMIT[Лимит 33% LZN]
        ORDER_LIMITS[Лимиты ордеров]
        MOA_SYSTEM[Система MOA]
        SLASHING[Штрафы и слэшинг]
        MONITORING[Мониторинг аномалий]
    end

    subgraph "Результат"
        SECURITY[Экономическая безопасность]
        FAIRNESS[Справедливость]
        STABILITY[Стабильность]
    end

    SYBIL --> ZKP
    CONCENTRATION --> LZN_LIMIT
    MARKET_MANIP --> ORDER_LIMITS
    MARKET_MANIP --> MONITORING
    NOTHING_AT_STAKE --> MOA_SYSTEM
    NOTHING_AT_STAKE --> SLASHING

    ZKP --> SECURITY
    LZN_LIMIT --> FAIRNESS
    ORDER_LIMITS --> STABILITY
    MOA_SYSTEM --> SECURITY
    SLASHING --> SECURITY
    MONITORING --> STABILITY

    style SECURITY fill:#90EE90
    style FAIRNESS fill:#FFB6C1
    style STABILITY fill:#87CEEB
```