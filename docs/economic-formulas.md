# Математические формулы экономической модели Volnix Protocol

## Базовые константы и параметры

### Системные константы
```
MAX_WRT_SUPPLY = 21,000,000 WRT
MAX_LZN_SUPPLY = 1,000,000 LZN
BASE_BLOCK_REWARD = 50 WRT
HALVING_INTERVAL = 210,000 блоков
MAX_VALIDATOR_LZN_SHARE = 0.33 (33%)
BASE_BLOCK_TIME = 6 секунд
MIN_BLOCK_TIME = 1 секунда
MAX_BLOCK_TIME = 60 секунд
```

### Настраиваемые параметры (через DAO)
```
MOA_COEFFICIENT = 0.001 (0.1%)
ANT_DAILY_ACCRUAL = 10 ANT
ANT_ACCUMULATION_LIMIT = 1000 ANT
AUCTION_DURATION = 1 час
ORDER_EXPIRY_TIME = 24 часа
```

## Формулы эмиссии и халвинга

### Расчет текущей награды за блок
```
current_reward = BASE_BLOCK_REWARD / (2^halving_count)

где:
halving_count = floor(block_number / HALVING_INTERVAL)
```

### Общая эмиссия WRT
```
total_emission = Σ(i=0 to ∞) reward_per_period_i

где:
reward_per_period_i = (BASE_BLOCK_REWARD / 2^i) × HALVING_INTERVAL
```

### Математический предел эмиссии
```
lim(n→∞) Σ(i=0 to n) (50 × 210,000) / 2^i = 50 × 210,000 × 2 = 21,000,000 WRT
```

## Формулы распределения доходов

### Пассивный доход валидатора (Контур 1)
```
validator_passive_income = (activated_lzn_validator / total_activated_lzn) × current_block_reward

где:
activated_lzn_validator = количество LZN, активированных валидатором
total_activated_lzn = общее количество активированных LZN в сети
```

### Доля валидатора в сети
```
validator_share = min(activated_lzn_validator / total_activated_lzn, MAX_VALIDATOR_LZN_SHARE)
```

### Ограничение максимальной активации LZN
```
max_lzn_per_validator = MAX_LZN_SUPPLY × MAX_VALIDATOR_LZN_SHARE
max_lzn_per_validator = 1,000,000 × 0.33 = 330,000 LZN
```

## Формулы внутреннего рынка ANT

### Базовая модель ценообразования
```
ant_price = base_price × (demand / supply) × activity_coefficient

где:
base_price = базовая цена ANT в WRT
demand = общий спрос на ANT
supply = общее предложение ANT
activity_coefficient = коэффициент сетевой активности
```

### Функция спроса на ANT
```
D(p) = α - β × p + γ × network_activity + δ × validator_count

где:
α = базовый спрос (константа)
β = эластичность спроса по цене
γ = чувствительность к сетевой активности
δ = чувствительность к количеству валидаторов
p = цена ANT в WRT
```

### Функция предложения ANT
```
S(p) = ε + ζ × p - η × accumulated_ant + θ × citizen_count

где:
ε = базовое предложение (константа)
ζ = эластичность предложения по цене
η = фактор истощения накопленных запасов
θ = влияние количества граждан
```

### Равновесная цена ANT
```
Условие равновесия: D(p*) = S(p*)

p* = (α - ε + γ × network_activity + δ × validator_count + η × accumulated_ant - θ × citizen_count) / (β + ζ)
```

### Динамическая корректировка цены
```
price_new = price_old × (1 + Δ_demand × adjustment_factor)

где:
Δ_demand = (buy_volume - sell_volume) / total_volume
adjustment_factor = 0.1 (10% максимальная корректировка за период)
```

## Формулы аукционов блоков

### Вероятность победы в аукционе
```
P(win_validator_i) = bid_validator_i / Σ(all_bids)

где:
bid_validator_i = ставка валидатора i в ANT
Σ(all_bids) = сумма всех ставок в аукционе
```

### Ожидаемая прибыль валидатора от аукциона
```
expected_profit = P(win) × expected_fees - ant_cost

где:
P(win) = вероятность победы
expected_fees = ожидаемые комиссии за блок
ant_cost = стоимость ANT для ставки
```

### Оптимальная стратегия ставок
```
optimal_bid = sqrt(expected_fees × total_competitor_bids)

Это решение уравнения максимизации ожидаемой прибыли:
max E[profit] = (bid / (bid + competitors)) × fees - ant_price × bid
```

### Расчет комиссий за блок
```
block_fees = Σ(transaction_fees) + priority_fees

где:
transaction_fees = базовые комиссии за транзакции
priority_fees = дополнительные комиссии за приоритет
```

## Формулы MOA (Минимальное Обязательство Активности)

### Расчет требуемого MOA
```
required_moa = activated_lzn × MOA_COEFFICIENT × epoch_duration

где:
activated_lzn = количество активированных LZN валидатором
MOA_COEFFICIENT = 0.001 (настраиваемый параметр)
epoch_duration = продолжительность эпохи в днях (обычно 7)
```

### Пример расчета MOA
```
Валидатор активировал: 100,000 LZN
MOA_COEFFICIENT: 0.001
Эпоха: 7 дней

required_moa = 100,000 × 0.001 × 7 = 700 ANT за неделю
daily_moa = 700 / 7 = 100 ANT в день
```

### Коэффициент соответствия MOA
```
moa_compliance = actual_moa / required_moa

где:
actual_moa = фактически использованные ANT за период
required_moa = требуемое количество ANT
```

### Система штрафов за нарушение MOA
```
penalty_multiplier = {
  1.0,    если moa_compliance >= 1.0    (без штрафа)
  1.0,    если 0.9 ≤ moa_compliance < 1.0    (предупреждение)
  0.75,   если 0.7 ≤ moa_compliance < 0.9    (штраф 25%)
  0.5,    если 0.5 ≤ moa_compliance < 0.7    (штраф 50%)
  0.0,    если moa_compliance < 0.5          (деактивация)
}

adjusted_reward = base_reward × penalty_multiplier
```

## Формулы динамического времени блока

### Расчет текущей активности сети
```
network_activity = Σ(ant_used_in_period) / period_duration

где:
ant_used_in_period = общее количество ANT, использованных за период
period_duration = продолжительность периода измерения
```

### Базовая активность (скользящее среднее)
```
base_activity = Σ(i=1 to 30) daily_activity_i / 30

где:
daily_activity_i = активность за день i
30 = период усреднения в днях
```

### Расчет времени блока
```
block_time = BASE_BLOCK_TIME × (base_activity / current_activity)^0.5

Ограничения:
block_time = max(MIN_BLOCK_TIME, min(MAX_BLOCK_TIME, block_time))
```

### Влияние на скорость халвинга
```
time_to_halving = blocks_remaining × average_block_time

где:
blocks_remaining = HALVING_INTERVAL - (current_block % HALVING_INTERVAL)
average_block_time = среднее время блока за последний период
```

## Формулы экономической безопасности

### Стоимость атаки 51%
```
attack_cost = 0.51 × total_activated_lzn × lzn_market_price + operational_costs

где:
total_activated_lzn = общее количество активированных LZN
lzn_market_price = рыночная цена LZN
operational_costs = операционные расходы на содержание узлов
```

### Коэффициент децентрализации Накамото
```
nakamoto_coefficient = min(k) такое что Σ(i=1 to k) validator_share_i > 0.33

где:
validator_share_i = доля валидатора i в общих активированных LZN
```

### Индекс концентрации Херфиндаля-Хиршмана (HHI)
```
HHI = Σ(i=1 to n) (validator_share_i × 100)^2

где:
validator_share_i = доля валидатора i (от 0 до 1)
n = общее количество валидаторов

Интерпретация:
HHI < 1500: низкая концентрация
1500 ≤ HHI < 2500: умеренная концентрация
HHI ≥ 2500: высокая концентрация
```

## Формулы оптимизации портфеля

### Целевая функция валидатора
```
max Σ(t=1 to T) [passive_income_t + active_income_t - costs_t] × (1 + r)^(-t)

где:
passive_income_t = пассивный доход в период t
active_income_t = активный доход в период t
costs_t = расходы в период t
r = ставка дисконтирования
T = горизонт планирования
```

### Ограничения оптимизации
```
Ограничения:
1. activated_lzn ≤ MAX_VALIDATOR_LZN_SHARE × MAX_LZN_SUPPLY
2. ant_purchases ≥ required_moa
3. available_capital ≥ lzn_activation_cost + ant_purchase_cost
4. operational_capacity ≥ minimum_infrastructure_requirements
```

### Оптимальное распределение капитала
```
Лагранжиан:
L = Σ profit_t × discount_t - λ₁(lzn_constraint) - λ₂(moa_constraint) - λ₃(capital_constraint)

Условия первого порядка:
∂L/∂lzn = marginal_profit_lzn - λ₁ = 0
∂L/∂ant = marginal_profit_ant - λ₂ = 0
```

## Формулы стабилизационных механизмов

### Буферный фонд
```
buffer_fund_target = total_wrt_supply × buffer_ratio

где:
buffer_ratio = 0.05 (5% от общего предложения WRT)
```

### Автоматическая корректировка параметров
```
new_parameter = old_parameter × (1 + adjustment × deviation_from_target)

где:
adjustment = коэффициент корректировки (обычно 0.01-0.05)
deviation_from_target = (actual_value - target_value) / target_value
```

### Индикатор волатильности рынка ANT
```
volatility = sqrt(Σ(i=1 to n) (price_return_i - mean_return)^2 / (n-1))

где:
price_return_i = ln(price_i / price_{i-1})
mean_return = среднее значение доходности
n = количество наблюдений
```

## Формулы анализа эффективности

### Коэффициент использования сети
```
network_utilization = actual_tps / theoretical_max_tps

где:
actual_tps = фактическое количество транзакций в секунду
theoretical_max_tps = теоретический максимум TPS
```

### Экономическая эффективность
```
economic_efficiency = total_value_created / total_resources_consumed

где:
total_value_created = общая экономическая ценность, созданная сетью
total_resources_consumed = общие ресурсы, потребленные сетью
```

### ROI для различных ролей
```
ROI_citizen = (ant_sales_revenue - verification_costs) / verification_costs
ROI_validator = (passive_income + active_income - total_costs) / total_investment
```

## Примеры численных расчетов

### Пример 1: Доходность валидатора
```
Дано:
- Активированные LZN: 100,000
- Общие активированные LZN: 500,000
- Текущая награда за блок: 25 WRT
- Блоков в день: 14,400 (6 сек/блок)
- Средние комиссии за блок: 5 WRT
- Вероятность выигрыша аукциона: 20%

Расчет:
Доля валидатора = 100,000 / 500,000 = 0.2 (20%)
Пассивный доход в день = 0.2 × 25 × 14,400 = 72,000 WRT
Активный доход в день = 0.2 × 5 × 14,400 = 14,400 WRT
Общий доход в день = 72,000 + 14,400 = 86,400 WRT
```

### Пример 2: Равновесная цена ANT
```
Дано:
- α (базовый спрос) = 1000
- β (эластичность спроса) = 50
- γ (чувствительность к активности) = 0.1
- Сетевая активность = 500 ANT/день
- ε (базовое предложение) = 200
- ζ (эластичность предложения) = 30

Расчет:
p* = (1000 - 200 + 0.1 × 500) / (50 + 30)
p* = (800 + 50) / 80 = 850 / 80 = 10.625 WRT за ANT
```

### Пример 3: Время до халвинга при разной активности
```
Дано:
- Блоков до халвинга: 50,000
- Высокая активность: 3 сек/блок
- Низкая активность: 12 сек/блок

Расчет:
При высокой активности: 50,000 × 3 = 150,000 сек = 1.74 дня
При низкой активности: 50,000 × 12 = 600,000 сек = 6.94 дня

Разница: 6.94 - 1.74 = 5.2 дня
```

Эти математические формулы обеспечивают точные расчеты всех экономических процессов в Volnix Protocol и служат основой для программной реализации экономической логики.