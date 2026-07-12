# FinHelper — Дизайн-спецификация референсов

> Полная документация 15 визуальных стилей для финансового приложения FinHelper.
> Каждый стиль разобран по единому шаблону: концепция, композиция, цвета, типографика, компоненты, глубина, анимации, UI Kit и prompt для генерации.

---

# Стиль №1 — Premium Dark Neon

## Общая концепция

Тёмный премиальный интерфейс с неоновыми акцентами фиолетово-синего спектра. Вызывает ощущение технологичности, премиальности и футуристичности. Напоминает современные криптобиржи и трейдинговые платформы уровня Binance Pro или Coinbase Advanced. Вызывает эмоции контроля, профессионализма и технологичности. Подходит для финтех-приложений, ориентированных на продвинутых пользователей, инвесторов и трейдеров. Напоминает интерфейсы: Linear, Vercel Dashboard, Raycast, Arc Browser.

## Общая композиция

- **Sidebar:** 240px, фиксированная слева, тёмный фон #0B0F1A с лёгким blur
- **Header:** Нет отдельного header — заголовок страницы «Обзор» расположен внутри контентной области
- **Сетка:** 12 колонок, контентная область с max-width ~1100px
- **Карточки:** Расположены в 2 колонки для метрик, 4 колонки для быстрых действий
- **Вертикальный rhythm:** 24px между секциями, 16px между карточками
- **Hero-блок:** Большая карточка «Общий баланс» с декоративным фоновым графиком, занимает всю ширину контента

## Цветовая палитра

```
Фон основной:        #0B1020
Фон sidebar:         #0B0F1A
Карточки:            #141A2D
Карточки hover:      #1B2340
Hero-блок:           #141A2D с gradient overlay
Primary:             #6E56CF (фиолетовый)
Primary glow:        rgba(110, 86, 207, 0.3)
Success:             #22C55E
Danger:              #F43F5E
Warning:             #F59E0B
Текст основной:      #FFFFFF
Текст вторичный:     #94A3B8
Текст третичный:     #64748B
Border:              rgba(255, 255, 255, 0.06)
Border hover:        rgba(255, 255, 255, 0.12)
Shadow:              0 4px 24px rgba(0, 0, 0, 0.4)
Shadow large:        0 20px 60px rgba(0, 0, 0, 0.5)
```

## Типографика

```
Шрифт:               Inter / SF Pro Display

Page title:          28px / 700 / -0.02em tracking
Section title:       18px / 600 / -0.01em
Card title:          14px / 600
Body:                13px / 400 / line-height 1.5
Caption:             11px / 500 / uppercase
Metric value:        32px / 700 / -0.02em
Metric label:        12px / 500 / text-secondary
Sidebar item:        13px / 500
Sidebar section:     11px / 600 / uppercase / text-tertiary
```

## Sidebar

- **Ширина:** 240px
- **Фон:** #0B0F1A, без границы справа, слёгка отделена фоном
- **Логотип:** «FinHelper» — 16px / 700, белый, с иконкой-эмблемой слева
- **Навигация:** Вертикальный список с иконками слева (20px), текст 13px/500
- **Иконки:** Линейные, strokeWidth 1.75, размер 20px
- **Active state:** Полупрозрачный фон rgba(110, 86, 207, 0.12), текст — primary цвет, иконка — primary
- **Hover:** rgba(255, 255, 255, 0.04)
- **Разделители:** Тонкая линия rgba(255,255,255,0.06) между секциями
- **Секции:** «Калькуляторы» — отдельная группа с заголовком uppercase 11px
- **Нижний элемент:** «Скрыть суммы» — toggle switch с иконкой глаза

## Header / Hero-блок

- **Нет传统ного header** — заголовок «Обзор» внутри контента
- **Hero-блок:** Полноширинная карточка с gradient background от #141A2D к #1E293B
- **Декоративный элемент:** Полупрозрачный линейный график (area chart) на фоне hero-блока, цвет — rgba(110, 86, 207, 0.2), обводка — rgba(110, 86, 207, 0.6)
- **Баланс:** Крупно «0,00 ₽» — 36px/700, белый
- **Подпись:** «Общий баланс» — 14px/500, text-secondary
- **Доходы/Расходы:** Справа в hero-блоке, 14px/500, с цветовыми индикаторами (зелёный/красный)
- **Подпись hero:** «0,00 ₽ за месяц» — 12px/400, text-tertiary
- **Blur:** backdrop-filter: blur(20px) на hero-блоке
- **Border:** 1px solid rgba(255,255,255,0.06)
- **Radius:** 20px

## Карточки

### Карточка «Баланс» (Hero)
- Размер: Полная ширина контента
- Padding: 28px 32px
- Background: gradient от #141A2D к #1E293B
- Декоративный area chart на фоне
- Border: 1px solid rgba(255,255,255,0.06)
- Radius: 20px
- Shadow: 0 8px 32px rgba(0,0,0,0.3)

### Карточки «Быстрые действия»
- Количество: 4 в ряд
- Размер: Равные, flex 1
- Padding: 20px
- Background: #141A2D
- Иконки: 40px контейнеры с gradient background (фиолетовый/синий)
- Radius: 16px
- Border: 1px solid rgba(255,255,255,0.06)
- Hover: фон слегка светлеет, border rgba(255,255,255,0.12)

### Карточки «Норма сбережений» / «Изменение за месяц»
- 2 в ряд
- Padding: 20px
- Background: #141A2D
- Radius: 16px
- Значение: 28px/700
- Label: 12px/500 text-secondary
- Border: 1px solid rgba(255,255,255,0.06)

### Карточки «Бюджет по категориям» / «Расходы по категориям»
- 2 в ряд
- Padding: 20px
- Background: #141A2D
- Radius: 16px
- Содержат placeholder для графиков
- Border: 1px solid rgba(255,255,255,0.06)

## Иконки

- **Стиль:** Линейные (outline)
- **Толщина линий:** 1.75px
- **Размер:** 20px в sidebar, 24px в карточках
- **Цвет:** #94A3B8 (вторичный), active — #6E56CF (primary)
- **Контейнер иконок быстрых действий:** 40px, скругление 12px, gradient background

## Графики

- **Тип:** Area chart (декоративный в hero), Bar chart (бюджет по категориям), Donut chart (расходы по категориям)
- **Толщина линии area chart:** 2px
- **Цвет:** rgba(110, 86, 207, 0.6) обводка, rgba(110, 86, 207, 0.1) заливка
- **Сетка:** Минималистичная, rgba(255,255,255,0.04)
- **Подписи:** 11px/500, text-tertiary

## Глубина интерфейса

**Elevation + Glass combination:**
- Карточки на фоне #0B1020 создают лёгкую глубину через тени
- Hero-блок имеет backdrop-filter: blur(20px) + saturate(150%)
- Sidebar — полупрозрачная с blur
- Общее ощущение: multi-layered dark UI с soft glow акцентами

## Свет

- **Источник:** сверху-слева
- **Цвет:** холодный синий
- **Интенсивность:** низкая
- **Glow:** на активных элементах и графиках, rgba(110, 86, 207, 0.3)
- **Ambient:** тёмный, с фиолетовым оттенком
- **Виньетка:** нет

## Blur

```
Sidebar:         backdrop-filter: blur(20px) saturate(150%)
Hero-блок:       backdrop-filter: blur(16px) saturate(140%)
Dropdown:        backdrop-filter: blur(24px) saturate(180%)
```

## Скругления

```
Cards:           20px
Hero card:       20px
Buttons:         12px
Icon containers: 12px
Quick actions:   16px
Charts:          16px
Inputs:          12px
Sidebar items:   10px
```

## Тени

```
Card shadow:        0 4px 16px rgba(0, 0, 0, 0.3)
Card shadow hover:  0 8px 32px rgba(0, 0, 0, 0.4)
Hero shadow:        0 8px 32px rgba(0, 0, 0, 0.3)
Sidebar shadow:     4px 0 24px rgba(0, 0, 0, 0.2)
Icon glow:          0 0 20px rgba(110, 86, 207, 0.3)
Inset (input):      inset 0 1px 2px rgba(0, 0, 0, 0.2)
```

## Анимации

- **Card hover:** transform: translateY(-2px), transition 200ms ease-out
- **Sidebar item hover:** background fade-in 150ms
- **Active state:** color transition 200ms
- **Quick action hover:** icon scale 1.05, glow increase 200ms
- **Chart animation:** draw-in effect на area chart, 600ms ease-out
- **Button press:** transform: scale(0.97), 100ms

## UI Kit

```
Button Primary:      bg #6E56CF, text white, radius 12px, hover lighten 10%
Button Secondary:    bg transparent, border rgba(255,255,255,0.12), radius 12px
Button Ghost:        bg transparent, text #94A3B8, hover rgba(255,255,255,0.04)
Button Icon:         40px, radius 12px, bg rgba(255,255,255,0.04)
Card:                bg #141A2D, border rgba(255,255,255,0.06), radius 20px
Metric:              value 28px/700, label 12px/500
Chart:               bg transparent, grid rgba(255,255,255,0.04)
Progress:            bg rgba(255,255,255,0.06), fill #6E56CF, radius 999px
Sidebar Item:        padding 10px 16px, radius 10px, icon 20px
Quick Action:        bg #141A2D, icon container 40px gradient, radius 16px
Toggle:              bg rgba(255,255,255,0.06), active #6E56CF
Badge:               bg rgba(110,86,207,0.15), text #6E56CF, radius 999px
```

## Дизайн-система

- **Grid:** 12 колонок, 24px gutter
- **Spacing scale:** 4, 8, 12, 16, 20, 24, 32, 40, 48, 64
- **Radius scale:** 8, 10, 12, 16, 20, 24
- **Shadow scale:** 3 уровня (sm, md, lg)
- **Typography scale:** 11, 12, 13, 14, 16, 18, 24, 28, 32, 36
- **Color tokens:** background, surface, surface-hover, border, text-primary, text-secondary, text-tertiary, primary, success, danger, warning

## Что обязательно сохранить

> Большой Hero-блок с декоративным графиком и крупным балансом — центральный элемент.
> Неоновые фиолетовые акценты на тёмном фоне — фирменноестиль.
> Мягкие glow-эффекты вокруг иконок и активных элементов.
> Минимум визуального шума, чёткая иерархия.
> Полупрозрачность и blur в sidebar и hero.

## Что нельзя менять

- Нельзя делать тяжёлые рамки и толстые границы
- Нельзя использовать яркие кислотные цвета
- Нельзя перегружать карточки деталями
- Нельзя убирать glow-эффекты — они создают премиальность
- Нельзя делать светлый фон — стиль строго тёмный

## Prompt для DeepSeek

> Создай modern desktop dashboard финансового приложения в стиле Premium Dark Neon. Используй тёмный фон #0B1020, карточки #141A2D с тонкими светящимися границами rgba(255,255,255,0.06), мягкие фиолетово-синие неоновые акценты #6E56CF с glow-эффектами. Hero-блок — полноширинная карточка с декоративным area chart на фоне и крупным балансом 36px. Sidebar 240px с полупрозрачным active состоянием и backdrop blur. Сетка 12 колонок, 24px gutter, радиусы 20px для карточек, 12px для кнопок. Иконки линейные 1.75px, размер 20px. Графики с плавными кривыми. Типографика Inter, минималистичная иерархия. Ощущение футуристичного банковского интерфейса уровня Linear + Vercel.

---

# Стиль №2 — Premium Light Organic

## Общая концепция

Светлый натуральный интерфейс с живыми декоративными элементами (листья, растения). Вызывает ощущение спокойствия, экологичности, заботы и уюта. Подходит для финтех-приложений, ориентированных на накопления, семейный бюджет, экологичные финансы. Напоминает: Aspiration, Some green banking apps, organic fintech.

## Общая композиция

- **Sidebar:** 240px, фиксированная слева, тёмно-зелёный фон #1A2E1A
- **Header:** Нет传统ного header
- **Сетка:** 12 колонок, контентная область ~1100px
- **Карточки:** 2 колонки для метрик, 4 для быстрых действий
- **Вертикальный rhythm:** 24px между секциями
- **Hero-блок:** Полноширинная карточка сsoft gradient и декоративными листьями

## Цветовая палитра

```
Фон основной:        #F5F3EE (warm off-white)
Фон sidebar:         #1A2E1A (тёмно-зелёный)
Карточки:            #FFFFFF
Карточки hover:      #FAFAF8
Hero-блок:           gradient от #E8F5E8 к #F0F7F0
Primary:             #2D7A3A (природный зелёный)
Primary light:       rgba(45, 122, 58, 0.1)
Success:             #22C55E
Danger:              #E53E3E
Warning:             #DD6B20
Текст основной:      #1A2E1A
Текст вторичный:     #5A6B5A
Текст третичный:     #8A9A8A
Border:              rgba(26, 46, 26, 0.08)
Border hover:        rgba(26, 46, 26, 0.15)
Shadow:              0 2px 12px rgba(26, 46, 26, 0.08)
Shadow large:        0 12px 40px rgba(26, 46, 26, 0.12)
```

## Типографика

```
Шрифт:               Inter / Nunito / DM Sans

Page title:          28px / 700
Section title:       18px / 600
Card title:          14px / 600
Body:                13px / 400 / line-height 1.6
Caption:             11px / 500
Metric value:        32px / 700
Metric label:        12px / 500
```

## Sidebar

- **Фон:** #1A2E1A (тёмно-зелёный)
- **Логотип:** «FinHelper» белый, 16px/700
- **Навигация:** Иконки 20px, текст 13px/500, белый/opacity 0.7
- **Active:** rgba(255,255,255,0.12), текст белый
- **Hover:** rgba(255,255,255,0.06)
- **Иконки:** Зеленоватые на активных, белые на обычных

## Header / Hero-блок

- **Hero:** Мягкий gradient от #E8F5E8 к #F0F7F0
- **Декоративные элементы:** Изображения листьев/растений в правом верхнем углу, opacity 0.3-0.5
- **Баланс:** 36px/700, тёмно-зелёный
- **График:** Мягкий area chart с gradient заливкой

## Карточки

- **Фон:** #FFFFFF
- **Border:** 1px solid rgba(26, 46, 26, 0.08)
- **Radius:** 20px
- **Shadow:** 0 2px 12px rgba(26, 46, 26, 0.08)
- **Padding:** 24px
- **Hover:** translateY(-1px), shadow increase

## Иконки

- **Стиль:** Rounded line
- **Толщина:** 1.75px
- **Размер:** 20px sidebar, 24px карточки
- **Цвет:** #5A6B5A, active — #2D7A3A

## Графики

- **Тип:** Area chart, Bar chart, Donut chart
- **Цвет:** #2D7A3A с gradient заливкой rgba(45, 122, 58, 0.15)
- **Сетка:** rgba(26, 46, 26, 0.06)

## Глубина интерфейса

**Soft + Organic:**
- Мягкие тени вместо границ
- Карточки парят над фоном
- Декоративные растительные элементы добавляют глубину
- Общее ощущение: natural, airy, breathing

## Свет

- Естественный рассеянный свет
- Направление: верх-лево
- Тёплые оттенки
- Нет резких теней

## Blur

Минимальный. Декоративные элементы могут иметь backdrop-filter: blur(8px).

## Скругления

```
Cards:           20px
Buttons:         14px
Icon containers: 14px
Charts:          16px
Inputs:          12px
```

## Тени

```
Card:            0 2px 12px rgba(26, 46, 26, 0.08)
Card hover:      0 8px 24px rgba(26, 46, 26, 0.12)
Sidebar:         4px 0 20px rgba(26, 46, 26, 0.15)
```

## Анимации

- **Hover:** Мягкое поднятие translateY(-1px), 200ms
- **Листья:** Лёгкое покачивание (CSS animation, 3-4s infinite)
- **Transition:** Всё через ease-out, без резких движений

## UI Kit

```
Button Primary:      bg #2D7A3A, text white, radius 14px
Button Secondary:    bg #E8F5E8, text #2D7A3A, radius 14px
Button Ghost:        bg transparent, text #5A6B5A
Card:                bg white, border rgba(26,46,26,0.08), radius 20px
Metric:              value 28px/700, label 12px/500
Progress:            bg rgba(45,122,58,0.1), fill #2D7A3A, radius 999px
Sidebar Item:        padding 10px 16px, radius 10px
Quick Action:        bg white, border, radius 16px, icon gradient green
Badge:               bg rgba(45,122,58,0.1), text #2D7A3A
```

## Дизайн-система

- **Grid:** 12 колонок, 24px gutter
- **Spacing:** 4, 8, 12, 16, 20, 24, 32, 40, 48
- **Color tokens:** warm-white, green-primary, green-light, text shades

## Что обязательно сохранить

> Декоративные растительные элементы — ключевая фишка.
> Тёплый natural фон #F5F3EE, не холодный белый.
> Ощущение покоя и заботы.
> Мягкие тени, никаких агрессивных границ.

## Что нельзя менять

- Нельзя делать холодные синие акценты
- Нельзя убирать декоративные листья
- Нельзя делать тёмный фон
- Нельзя ставить толстые рамки

## Prompt для DeepSeek

> Создай desktop dashboard финансового приложения в стиле Premium Light Organic. Тёплый фон #F5F3EE, карточки белые #FFFFFF с мягкими тенями rgba(26,46,26,0.08). Primary — природный зелёный #2D7A3A. Hero-блок сsoft gradient от #E8F5E8 к #F0F7F0 и декоративными полупрозрачными листьями. Sidebar 240px тёмно-зелёный #1A2E1A. Радиусы 20px, иконки rounded line 1.75px. Графики с зелёными gradient заливками. Ощущение спокойствия, экологичности, заботы. Стиль для приложения по накоплениям и семейному бюджету.

---

# Стиль №3 — Dark Dashboard v2

## Общая концепция

Минималистичный тёмный дашборд без лишних декоративных элементов. Чистая функциональность, строгая иерархия, профессиональный вид. Подходит для серьёзных финтех-инструментов, аналитических платформ. Напоминает: Grafana, Datadog, Stripe Dashboard.

## Общая композиция

- **Sidebar:** 220px, тёмный фон
- **Сетка:** 12 колонок
- **Карточки:** Плотная компоновка, 2-3 колонки
- **Hero:** Строгий, без декоративных элементов

## Цветовая палитра

```
Фон:                 #0F1117
Sidebar:             #0A0C10
Карточки:            #1A1D27
Карточки hover:      #22262F
Primary:             #3B82F6 (синий)
Success:             #22C55E
Danger:              #EF4444
Текст основной:      #F1F5F9
Текст вторичный:     #94A3B8
Текст третичный:     #64748B
Border:              rgba(255, 255, 255, 0.06)
Shadow:              0 2px 8px rgba(0, 0, 0, 0.3)
```

## Типографика

```
Шрифт:               Inter / JetBrains Mono (для чисел)

Page title:          24px / 600
Card title:          13px / 500
Body:                13px / 400
Caption:             11px / 500
Metric value:        28px / 700
```

## Sidebar

- 220px, строгая вертикальная навигация
- Иконки 18px, текст 13px
- Active: левая border 2px primary, фон rgba(59, 130, 246, 0.08)
- Без декораций

## Карточки

- **Фон:** #1A1D27
- **Border:** 1px solid rgba(255,255,255,0.06)
- **Radius:** 12px (меньше чем в других стилях)
- **Padding:** 20px
- **Компактная компоновка**

## Графики

- **Стиль:** Stroke-based, без заливки
- **Толщина:** 1.5px
- **Сетка:** rgba(255,255,255,0.04)

## Глубина

**Flat + Elevation:**
- Минимум теней, больше границ
- Чёткая структура через разделители
- Professional, data-focused

## Скругления

```
Cards:     12px
Buttons:   8px
Inputs:    8px
```

## Prompt для DeepSeek

> Minimalist dark dashboard для финансового приложения. Фон #0F1117, карточки #1A1D27, синий акцент #3B82F6. Sidebar 220px строгий. Radius 12px, компактная компоновка. Интерфейс для аналитиков и профессионалов. Чистая функциональность без декораций.

---

# Стиль №4 — Terminal / Hacker Green

## Общая концепция

Ретро-терминальный стиль в духе старых компьютерных систем. Монохромный зелёный на чёрном фоне, моноширинный шрифт, текстовый интерфейс. Вызывает ощущение хакерской эстетики, ностальгии, технической мощи. Подходит для технических финтех-инструментов, криптовалютных платформ для продвинутых пользователей. Напоминает: Terminal, hacker aesthetic, retro computing.

## Общая композиция

- **Sidebar:** Текстовая, без иконок, только текстовые ссылки с `>` маркерами
- **Сетка:** Текстовая раскладка, табличная структура
- **Нет traditional карточек** — всё через рамки и текст
- **Компоновка:** Строгая, как в терминале

## Цветовая палитра

```
Фон:                 #0A0A0A
Текст основной:      #00FF41 (matrix green)
Текст вторичный:     #00CC33
Текст третичный:     #009926
Border:              #00FF41 (точечные рамки)
Success:             #00FF41
Danger:              #FF0000
Warning:             #FFFF00
Shadow:              0 0 10px rgba(0, 255, 65, 0.2)
Glow:                0 0 20px rgba(0, 255, 65, 0.3)
```

## Типографика

```
Шрифт:               JetBrains Mono / Fira Code / Source Code Pro

Заголовки:           16px / 700 / uppercase
Подзаголовки:        14px / 600 / uppercase
Body:                13px / 400
Caption:             11px / 400
Метрики:             20px / 700
```

## Sidebar

- **Нет traditional sidebar** — текстовое меню
- Формат: `> ГЛАВНАЯ`, `> ОПЕРАЦИИ`, `> ЦЕЛИ`
- Активный: brighter green, без маркера `>`
- Подменю: отступ 2 spaces

## Header

- **Формат:** Текстовый заголовок + timestamp
- Пример: `ОБЗОР СИСТЕМЫ          2024-01-01 12:00:00`
- Всё uppercase, моноширинный шрифт

## Карточки

- **Нет traditional карточек** — блоки разделены текстовыми рамками
- Рамки: `+---+---+` формат или тонкие border
- Заголовки блоков: `БЫСТРЫЕ ДЕЙСТВИЯ:`
- Значения: `0,00 Р`

## Иконки

- **Нет traditional иконок** — текстовые символы: `+`, `◆`, `■`, `□`
- Или ASCII art

## Графики

- ASCII-графики или text-based bar charts
- Пример: `████████░░░░ 65%`

## Глубина

**Flat / Terminal:**
- Абсолютно плоский
- Глубина только через glow-эффекты
- Монохромная палитра

## Свет

- Glow-эффекты от зелёного текста
- Ambient: rgba(0, 255, 65, 0.05) на фоне
- Виньетка: возможна тёмная по краям

## Blur

Нет. Абсолютно чёткий растеризованный вид.

## Скругления

```
Все элементы:  0px (абсолютно острые углы)
```

## Тени

```
Text glow:     0 0 10px rgba(0, 255, 65, 0.3)
Card glow:     0 0 20px rgba(0, 255, 65, 0.1)
```

## Анимации

- **Terminal typing effect:** символы появляются по одному
- **Blinking cursor:** мигающий курсор
- **Scanlines:** лёгкий CRT-эффект (опционально)
- **Glow pulse:** пульсация glow 2-3s infinite

## UI Kit

```
Button:         border 1px solid #00FF41, bg transparent, text #00FF41
Button active:  bg rgba(0, 255, 65, 0.1), glow
Input:          border 1px solid #00FF41, bg transparent, caret #00FF41
Card:           border 1px solid #00FF41, bg transparent
Toggle:         [ON] / [OFF] текст
Badge:          [ACTIVE] текстовый
```

## Prompt для DeepSeek

> Terminal-style финансовый дашборд. Чёрный фон #0A0A0A, зелёный текст #00FF41 с glow-эффектом. Моноширинный шрифт JetBrains Mono. Текстовые рамки вместо карточек. Sidebar — текстовое меню с `>` маркерами. ASCII-графики. Всё uppercase. Ощущение хакерской терминальной эстетики и ностальгии по ретро-компьютерам. Нулевые скругления, абсолютный flat.

---

# Стиль №5 — Soft Glass Pastel

## Общая концепция

Мягкий пастельный интерфейс с элементами стеклянности (glassmorphism). Нежные розовые, фиолетовые и голубые оттенки. Вызывает ощущение мягкости, 접근ности, дружелюбия. Подходит для финтех-приложений для широкой аудитории, особенно для женщин и молодёжи. Напоминает: Robinhood, Cash App, современные необанки.

## Общая композиция

- **Sidebar:** 240px, полупрозрачный с blur
- **Сетка:** 12 колонок
- **Карточки:** Мягкие, с glassmorphism
- **Hero:** Пастельный gradient с blur

## Цветовая палитра

```
Фон:                 #F8F5FF (лавандовый off-white)
Sidebar:             rgba(255, 255, 255, 0.7) + blur
Карточки:            rgba(255, 255, 255, 0.6) + blur
Primary:             #A78BFA (мягкий фиолетовый)
Primary light:       #C4B5FD
Secondary:           #F472B6 (мягкий розовый)
Success:             #34D399
Danger:              #FB7185
Текст основной:      #1E1B4B
Текст вторичный:     #6B7280
Border:              rgba(255, 255, 255, 0.5)
Shadow:              0 4px 24px rgba(167, 139, 250, 0.15)
```

## Типографика

```
Шрифт:               Inter / Poppins

Page title:          28px / 700
Card title:          14px / 600
Body:                13px / 400
Caption:             11px / 500
Metric value:        32px / 700
```

## Sidebar

- **Background:** rgba(255, 255, 255, 0.7)
- **Backdrop-filter:** blur(20px) saturate(180%)
- **Active:** rgba(167, 139, 250, 0.15), text #A78BFA
- **Hover:** rgba(0, 0, 0, 0.03)

## Hero-блок

- **Background:** Gradient от #EDE9FE к #FCE7F3
- **Blur:** backdrop-filter: blur(16px)
- **Border:** 1px solid rgba(255,255,255,0.6)
- **Декоративные элементы:** Мягкие gradient blobs

## Карточки

- **Background:** rgba(255, 255, 255, 0.6)
- **Backdrop-filter:** blur(12px) saturate(150%)
- **Border:** 1px solid rgba(255,255,255,0.5)
- **Radius:** 24px (большие скругления)
- **Shadow:** 0 4px 24px rgba(167, 139, 250, 0.1)
- **Padding:** 24px

## Иконки

- **Стиль:** Rounded, мягкие
- **Толщина:** 1.75px
- **Размер:** 20px
- **Контейнеры:** Пастельные gradient фоны

## Графики

- **Стиль:** Мягкие gradient заливки
- **Цвета:** Пастельные оттенки
- **Сетка:** Минимальная

## Глубина

**Glassmorphism:**
- Многослойность через blur и прозрачность
- Фон просвечивает через карточки
- Ощущение воздушности и лёгкости

## Свет

- Мягкий рассеянный
- Pastel gradient blobs как декоративные элементы
- Тёплые оттенки

## Blur

```
Sidebar:     backdrop-filter: blur(20px) saturate(180%)
Cards:       backdrop-filter: blur(12px) saturate(150%)
Hero:        backdrop-filter: blur(16px) saturate(160%)
Dropdown:    backdrop-filter: blur(24px) saturate(200%)
```

## Скругления

```
Cards:       24px
Buttons:     16px
Icon cont.:  16px
Inputs:      14px
Charts:      20px
```

## Prompt для DeepSeek

> Soft Glassmorphism dashboard. Фон #F8F5FF, карточки rgba(255,255,255,0.6) с backdrop blur. Пастельные акценты: фиолетовый #A78BFA, розовый #F472B6. Hero-блок с gradient blobs. Sidebar полупрозрачный. Радиусы 24px. Мягкие тени. Ощущение нежности, 접근ности, дружелюбия. Для широкой аудитории.

---

# Стиль №6 — Muted Light Premium

## Общая концепция

Приглушённый светлый премиальный стиль. Сдержанные цвета, элегантная типографика, утончённые акценты. Напоминает: премиальные банковские интерфейсы, private banking, Coutts, HSBC Private Bank. Подходит для состоятельных клиентов, консервативных финтех-продуктов. Вызывает ощущение надёжности, сдержанной роскоши, доверия.

## Общая композиция

- **Sidebar:** 220px, белый/cream фон с мягким border справа (1px solid rgba(0,0,0,0.06))
- **Header:** Нет传统ного hero-блока — заголовок «Обзор» + period selector справа
- **Сетка:** 12 колонок, просторная компоновка, 24px gutter
- **Карточки:** 2 колонки для метрик, 4 для быстрых действий (иконки с текстом)
- **Вертикальный rhythm:** 28px между секциями, 16px между карточками
- **Hero-блок:** Нет large hero — баланс показан как обычный metric в карточке

## Цветовая палитра

```
Фон основной:        #FAFAF9 (warm grey off-white)
Фон sidebar:         #FFFFFF
Карточки:            #FFFFFF
Карточки hover:      #F5F5F4
Primary:             #7C3AED (приглушённый фиолетовый)
Primary light:       rgba(124, 58, 237, 0.08)
Success:             #16A34A
Danger:              #DC2626
Warning:             #CA8A04
Текст основной:      #1C1917
Текст вторичный:     #78716C
Текст третичный:     #A8A29E
Border:              rgba(0, 0, 0, 0.06)
Border hover:        rgba(0, 0, 0, 0.1)
Shadow:              0 1px 3px rgba(0, 0, 0, 0.06)
Shadow medium:       0 4px 12px rgba(0, 0, 0, 0.08)
```

## Типографика

```
Шрифт основной:      Inter
Шрифт заголовков:    Inter (semibold, без decorative шрифтов)

Page title:          26px / 600 / -0.01em
Section title:       16px / 600
Card title:          14px / 500
Body:                13px / 400 / line-height 1.5
Caption:             11px / 500 / text-tertiary
Metric value:        30px / 700 / -0.01em
Metric label:        12px / 500 / text-secondary
Sidebar item:        13px / 500
Sidebar section:     11px / 600 / uppercase / text-tertiary
Period selector:     13px / 500, с иконкой chevron
```

## Sidebar

- **Ширина:** 220px
- **Фон:** #FFFFFF, border-right: 1px solid rgba(0,0,0,0.06)
- **Логотип:** «FinHelper» — 15px / 700, #1C1917, с иконкой слева
- **Навигация:** Вертикальный список, иконки 18px strokeWidth 1.75, текст 13px/500
- **Active state:** Левая border 2px #7C3AED, фон rgba(124, 58, 237, 0.05), текст #7C3AED
- **Hover:** rgba(0, 0, 0, 0.03)
- **Разделители:** Тонкая линия rgba(0,0,0,0.06) между группами
- **Секции:** «Калькуляторы» — подзаголовок 11px/600 uppercase, #A8A29E
- **Нижний элемент:** «Скрыть суммы» — toggle с иконкой глаза

## Header / Hero-блок

- **Нет传统ного hero** — заголовок «Обзор» 26px/600, слева
- **Period selector:** Справа в шапке контента, «Период: Месяц» с dropdown chevron
- **Баланс:** Показан в карточке «Баланс» — 30px/700, #1C1917
- **Доходы/Расходы:** Отдельные карточки с числовыми значениями

## Карточки

### Карточка «Баланс»
- Размер: ~33% ширины контента
- Padding: 20px
- Background: #FFFFFF
- Border: 1px solid rgba(0,0,0,0.06)
- Radius: 16px
- Shadow: 0 1px 3px rgba(0,0,0,0.06)
- Значение: 30px/700, #1C1917
- Label: «Баланс» 13px/500, #78716C

### Карточки «Доходы» / «Расходы»
- Аналогичные «Балансу», но с цветовыми индикаторами
- Доходы: #16A34A, Расходы: #DC2626

### Карточки «Быстрые действия»
- 4 в ряд, горизонтальные кнопки
- Padding: 16px
- Background: #FFFFFF
- Иконки: 36px контейнеры сsoft background
- Radius: 12px
- Border: 1px solid rgba(0,0,0,0.06)
- Hover: background #F5F5F4, border rgba(0,0,0,0.1)

### Карточки «Норма сбережений» / «Изменение за месяц»
- 2 в ряд
- Padding: 20px
- Radius: 16px
- Значение: 28px/700
- Label: 12px/500 #78716C

### Карточки «Бюджет по категориям» / «Расходы по категориям»
- 2 в ряд
- Padding: 20px
- Radius: 16px
- Placeholder для графиков

## Иконки

- **Стиль:** Rounded line
- **Толщина линий:** 1.75px
- **Размер:** 18px в sidebar, 22px в карточках
- **Цвет:** #78716C (вторичный), active — #7C3AED (primary)
- **Контейнеры быстрых действий:** 36px, скругление 10px, фон rgba(124, 58, 237, 0.06)

## Графики

- **Тип:** Bar chart (бюджет), Donut chart (расходы)
- **Цвета:** Приглушённые тона — #7C3AED, #A78BFA, #C4B5FD
- **Сетка:** rgba(0, 0, 0, 0.04)
- **Подписи:** 11px/500, #A8A29E

## Глубина интерфейса

**Soft + Flat:**
- Минимальные тени (только sm level)
- Разделение через border, не тени
- Белые карточки на warm grey фоне
- Ощущение: сдержанная элегантность,纸质质感

## Свет

- Равномерный рассеянный свет
- Без directional shadows
- Тёплые undertone в фоне
- Нет glow-эффектов

## Blur

Не используется. Полностью непрозрачный интерфейс.

## Скругления

```
Cards:           16px
Buttons:         12px
Icon containers: 10px
Charts:          12px
Inputs:          10px
Sidebar items:   8px
```

## Тени

```
Card:            0 1px 3px rgba(0, 0, 0, 0.06)
Card hover:      0 4px 12px rgba(0, 0, 0, 0.08)
Dropdown:        0 8px 24px rgba(0, 0, 0, 0.12)
```

## Анимации

- **Card hover:** translateY(-1px), shadow increase, 200ms ease-out
- **Sidebar item:** background fade 150ms
- **Transition:** Все переходы через ease-out, без резких движений
- **Button press:** scale(0.98), 100ms

## UI Kit

```
Button Primary:      bg #7C3AED, text white, radius 12px, hover darken 10%
Button Secondary:    bg rgba(124,58,237,0.08), text #7C3AED, radius 12px
Button Ghost:        bg transparent, text #78716C, hover rgba(0,0,0,0.03)
Button Icon:         36px, radius 10px, bg rgba(124,58,237,0.06)
Card:                bg white, border rgba(0,0,0,0.06), radius 16px
Metric:              value 30px/700, label 12px/500
Progress:            bg rgba(124,58,237,0.1), fill #7C3AED, radius 999px
Sidebar Item:        padding 8px 12px, radius 8px, left-border active
Quick Action:        bg white, border, radius 12px, icon 36px
Badge:               bg rgba(124,58,237,0.08), text #7C3AED, radius 999px
Toggle:              bg rgba(0,0,0,0.1), active #7C3AED
Period Selector:     bg white, border, radius 8px, chevron icon
```

## Дизайн-система

- **Grid:** 12 колонок, 24px gutter
- **Spacing scale:** 4, 8, 12, 16, 20, 24, 28, 32, 40, 48
- **Radius scale:** 6, 8, 10, 12, 16, 20
- **Shadow scale:** 2 уровня (sm, md)
- **Typography scale:** 11, 12, 13, 14, 16, 18, 22, 26, 30
- **Color tokens:** surface, surface-hover, border, text-primary, text-secondary, text-tertiary, primary, primary-light, success, danger

## Что обязательно сохранить

> Сдержанность и элегантность — ключевые слова.
> Приглушённые цвета, никаких ярких акцентов.
> Просторная компоновка,white space как design element.
> Типографика — главный носитель информации.
> Warm undertone в фоне, не холодный белый.

## Что нельзя менять

- Нельзя делать яркие неоновые акценты
- Нельзя перегружать карточки деталями
- Нельзя убирать white space
- Нельзя делать холодные серые фоны
- Нельзя ставить толстые рамки

## Prompt для DeepSeek

> Создай desktop dashboard финансового приложения в стиле Muted Light Premium. Тёплый белый фон #FAFAF9, карточки белые #FFFFFF с тонкими border rgba(0,0,0,0.06) и минимальными тенями. Приглушённый фиолетовый акцент #7C3AED. Sidebar 220px белый с border справа, active — левая border 2px. Просторная компоновка, 24px gutter, 12 колонок. Радиусы 16px для карточек, 12px для кнопок. Иконки rounded line 1.75px. Типографика Inter, элегантная иерархия. Period selector справа. Ощущение сдержанной роскоши, надёжности и доверия. Для премиальных банковских интерфейсов уровня private banking.

---

# Стиль №7 — Stripe Inspired

## Общая концепция

Тёмный интерфейс в стиле Stripe — чистый, современный, с фирменным gradient hero-блоком. Технологичный вид, professional. Вызывает ощущение инновационности, технологической мощи, премиальности. Напоминает напрямую: Stripe Dashboard, Stripe Radar, Stripe Atlas. Подходит для серьёзных финтех-платформ, платёжных систем, B2B的产品.

## Общая композиция

- **Sidebar:** 240px, тёмный фон, чистая навигация
- **Header:** Минималистичный, заголовок «Обзор» + period selector
- **Сетка:** 12 колонок, контентная область ~1100px
- **Карточки:** 3 колонки для метрик, 4 для быстрых действий
- **Вертикальный rhythm:** 24px между секциями, 16px между карточками
- **Hero-блок:** Крупная gradient карточка с балансом и декоративными blobs

## Цветовая палитра

```
Фон основной:        #0A0E27 (deep navy)
Фон sidebar:         #0D1117
Карточки:            #161B22
Карточки hover:      #1C2128
Hero gradient:       #635BFF → #7C3AED → #A855F7 (multi-stop)
Primary:             #635BFF (Stripe purple)
Primary light:       #818CF8
Success:             #30D158
Danger:              #FF3B30
Warning:             #FFB800
Текст основной:      #F0F6FC
Текст вторичный:     #8B949E
Текст третичный:     #6E7681
Border:              rgba(255, 255, 255, 0.08)
Border hover:        rgba(255, 255, 255, 0.14)
Shadow:              0 2px 8px rgba(0, 0, 0, 0.3)
Shadow large:        0 12px 40px rgba(0, 0, 0, 0.4)
```

## Типографика

```
Шрифт:               Inter / -apple-system

Page title:          28px / 700 / -0.02em
Section title:       16px / 600
Card title:          14px / 500
Body:                14px / 400 / line-height 1.5
Caption:             12px / 500
Metric value:        36px / 700 / -0.02em
Metric label:        13px / 500 / text-secondary
Sidebar item:        13px / 500
```

## Sidebar

- **Ширина:** 240px
- **Фон:** #0D1117, border-right: 1px solid rgba(255,255,255,0.06)
- **Логотип:** «FinHelper» — 16px / 700, белый
- **Навигация:** Иконки 20px strokeWidth 1.75, текст 13px/500
- **Active:** Фон rgba(99, 91, 255, 0.12), текст #818CF8, иконка #818CF8
- **Hover:** rgba(255, 255, 255, 0.04)
- **Разделители:** rgba(255,255,255,0.06) между секциями

## Header / Hero-блок

- **Hero-блок:** Полноширинная карточка с multi-stop gradient #635BFF → #7C3AED → #A855F7
- **Декоративные элементы:** Мягкие gradient blobs (radial gradients) на фоне hero, создающие mesh-like эффект
- **Баланс:** Крупно «0,00 ₽» — 36px/700, белый, с лёгким text-shadow glow
- **Label:** «Общий баланс» — 14px/500, rgba(255,255,255,0.8)
- **Доходы/Расходы:** Справа, 14px/500, с цветовыми индикаторами
- **Radius:** 20px
- **Border:** Нет (gradient сам по себе является границей)

## Карточки

### Карточка «Баланс» (Hero)
- Размер: Полная ширина
- Padding: 32px
- Background: Multi-stop gradient
- Декоративные blobs на фоне
- Radius: 20px
- Без border (gradient)

### Карточки «Быстрые действия»
- 4 в ряд
- Padding: 16px
- Background: #161B22
- Border: 1px solid rgba(255,255,255,0.08)
- Radius: 16px
- Иконки: 40px контейнеры с gradient background
- Hover: border rgba(255,255,255,0.14), translateY(-1px)

### Карточки метрик
- 2-3 в ряд
- Padding: 20px
- Background: #161B22
- Border: 1px solid rgba(255,255,255,0.08)
- Radius: 16px
- Значение: 28px/700
- Label: 12px/500 text-secondary

### Карточки графиков
- 2 в ряд
- Padding: 20px
- Background: #161B22
- Border: 1px solid rgba(255,255,255,0.08)
- Radius: 16px

## Иконки

- **Стиль:** Rounded line
- **Толщина:** 1.75px
- **Размер:** 20px sidebar, 24px карточки
- **Цвет:** #8B949E, active — #818CF8
- **Контейнеры быстрых действий:** 40px, gradient background #635BFF → #7C3AED

## Графики

- **Тип:** Area chart (hero), Bar chart, Donut chart
- **Цвета:** #635BFF, #818CF8, #A78BFA (градиенты фиолетового)
- **Сетка:** rgba(255,255,255,0.04)
- **Подписи:** 11px/500, #6E7681
- **Обводка:** 2px, #635BFF
- **Заливка:** Gradient rgba(99, 91, 255, 0.2) → transparent

## Глубина интерфейса

**Elevation + Gradient:**
- Hero-блок — яркий gradient, создаёт визуальный фокус
- Карточки — тёмные с тонкими border, лёгкие тени
- Sidebar — отдельный слой с border
- Ощущение: tech-forward, premium, innovative

## Свет

- Gradient blobs в hero создают ambient glow
- Primary glow: rgba(99, 91, 255, 0.15)
- Виньетка: возможна тёмная по краям hero
- Общая интенсивность: medium

## Blur

```
Hero-блок:       backdrop-filter: blur(12px) saturate(150%) (опционально)
Dropdown:        backdrop-filter: blur(20px) saturate(180%)
```

## Скругления

```
Cards:           16px
Hero card:       20px
Buttons:         12px
Icon containers: 12px
Charts:          16px
Inputs:          10px
```

## Тени

```
Card:            0 2px 8px rgba(0, 0, 0, 0.3)
Card hover:      0 8px 24px rgba(0, 0, 0, 0.4)
Hero:            0 12px 40px rgba(0, 0, 0, 0.3)
Dropdown:        0 12px 40px rgba(0, 0, 0, 0.5)
```

## Анимации

- **Hero gradient:** Медленное движение gradient blobs (10-15s infinite, ease-in-out)
- **Card hover:** translateY(-2px), border brighten, 200ms ease-out
- **Quick action hover:** Icon glow increase, 200ms
- **Chart animation:** Draw-in effect, 800ms ease-out
- **Button press:** scale(0.97), 100ms

## UI Kit

```
Button Primary:      bg #635BFF, text white, radius 12px, hover #7C3AED
Button Secondary:    bg rgba(99,91,255,0.12), text #818CF8, radius 12px
Button Ghost:        bg transparent, text #8B949E, hover rgba(255,255,255,0.04)
Button Icon:         40px, radius 12px, bg rgba(99,91,255,0.1)
Card:                bg #161B22, border rgba(255,255,255,0.08), radius 16px
Metric:              value 28px/700, label 12px/500
Chart:               bg transparent, grid rgba(255,255,255,0.04)
Progress:            bg rgba(99,91,255,0.15), fill #635BFF, radius 999px
Sidebar Item:        padding 10px 16px, radius 10px
Quick Action:        bg #161B22, border, icon 40px gradient, radius 16px
Badge:               bg rgba(99,91,255,0.15), text #818CF8, radius 999px
Toggle:              bg rgba(255,255,255,0.08), active #635BFF
Period Selector:     bg #161B22, border, radius 8px
```

## Дизайн-система

- **Grid:** 12 колонок, 24px gutter
- **Spacing scale:** 4, 8, 12, 16, 20, 24, 32, 40, 48
- **Radius scale:** 8, 10, 12, 16, 20, 24
- **Shadow scale:** 3 уровня (sm, md, lg)
- **Typography scale:** 12, 13, 14, 16, 20, 24, 28, 32, 36
- **Color tokens:** background, surface, surface-hover, border, text, text-secondary, text-tertiary, primary, primary-gradient, success, danger

## Что обязательно сохранить

> Gradient hero-блок — фирменная фишка Stripe-стиля.
> Multi-stop gradient #635BFF → #A855F7 обязательно.
> Gradient blobs на фоне hero — mesh-like декор.
> Крупный баланс с glow на gradient фоне.
> Технологичный, professional вид.

## Что нельзя менять

- Нельзя убирать gradient hero
- Нельзя делать светлый фон
- Нельзя ставить толстые рамки
- Нельзя использовать яркие нефиолетовые акценты
- Нельзя перегружать карточки — минимализм ключевой

## Prompt для DeepSeek

> Создай desktop dashboard финансового приложения в стиле Stripe Inspired. Тёмный фон #0A0E27, карточки #161B22 с тонкими border rgba(255,255,255,0.08). Hero-блок — полноширинная карточка с multi-stop gradient #635BFF → #7C3AED → #A855F7 и декоративными gradient blobs на фоне. Баланс крупный 36px белый с glow. Sidebar 240px тёмный. Радиусы 16px для карточек, 12px для кнопок. Иконки rounded line 1.75px, контейнеры быстрых действий с gradient background. Типографика Inter. Графики с фиолетовыми gradient заливками. Ощущение технологичности, инновационности, премиальности. Для серьёзных финтех-платформ и платёжных систем уровня Stripe.

---

# Стиль №8 — Linear Dark

## Общая концепция

Минималистичный тёмный интерфейс в стиле Linear. Абсолютный минимум визуальных элементов, акцент на контент и生产力. Вызывает ощущение чистоты, фокуса, профессионализма. Напоминает: Linear, Raycast, Vercel, Notion Dark. Подходит для productivity-oriented финтех-инструментов, для тех, кто ценит чистоту и функциональность.

## Общая композиция

- **Sidebar:** 240px, ultra-clean, minimal decoration
- **Header:** Заголовок «Обзор» + actions справа
- **Сетка:** 12 колонок, плотная компоновка
- **Карточки:** Компактные, minimal padding
- **Вертикальный rhythm:** 20px между секциями, 12px между карточками
- **Hero-блок:** Сдержанный, без gradient/decoration

## Цветовая палитра

```
Фон основной:        #0D0D0D (near black)
Фон sidebar:         #111111
Карточки:            #1A1A1A
Карточки hover:      #222222
Primary:             #5E6AD2 (Linear purple)
Primary light:       rgba(94, 106, 210, 0.15)
Success:             #2DA44E
Danger:              #CF222E
Warning:             #BF8700
Текст основной:      #E6E6E6
Текст вторичный:     #8A8A8A
Текст третичный:     #555555
Border:              rgba(255, 255, 255, 0.06)
Border hover:        rgba(255, 255, 255, 0.1)
Shadow:              0 1px 2px rgba(0, 0, 0, 0.2)
```

## Типографика

```
Шрифт:               Inter

Page title:          24px / 600 / -0.01em
Section title:       15px / 600
Card title:          13px / 500
Body:                13px / 400 / line-height 1.5
Caption:             12px / 400 / text-tertiary
Metric value:        28px / 600 / -0.01em
Metric label:        12px / 400 / text-secondary
Sidebar item:        13px / 400
```

## Sidebar

- **Ширина:** 240px
- **Фон:** #111111, border-right: 1px solid rgba(255,255,255,0.06)
- **Логотип:** «FinHelper» — 15px / 600, #E6E6E6
- **Навигация:** Иконки 18px strokeWidth 1.5, текст 13px/400
- **Active:** Фон rgba(94, 106, 210, 0.08), текст #E6E6E6, иконка #5E6AD2
- **Hover:** rgba(255, 255, 255, 0.04)
- **Без разделителей** — чистая навигация
- **Компактная компоновка:** padding 8px 12px для items

## Header / Hero-блок

- **Нет traditional hero** — заголовок «Обзор» 24px/600
- **Actions:** Period selector справа, minimal design
- **Баланс:** 28px/600, #E6E6E6 — не выделяется hero-блоком
- **Overall:** ultra-functional, data-first approach

## Карточки

### Карточка «Баланс»
- Размер: Полная ширина или 50%
- Padding: 16px
- Background: #1A1A1A
- Border: 1px solid rgba(255,255,255,0.06)
- Radius: 8px
- Без shadow
- Значение: 28px/600
- Label: 12px/400 #8A8A8A

### Карточки «Быстрые действия»
- 4 в ряд
- Padding: 12px
- Background: #1A1A1A
- Border: 1px solid rgba(255,255,255,0.06)
- Radius: 8px
- Иконки: 32px контейнеры
- Hover: border rgba(255,255,255,0.1)

### Карточки метрик
- 2 в ряд
- Padding: 16px
- Background: #1A1A1A
- Border: 1px solid rgba(255,255,255,0.06)
- Radius: 8px

### Карточки графиков
- 2 в ряд
- Padding: 16px
- Background: #1A1A1A
- Border: 1px solid rgba(255,255,255,0.06)
- Radius: 8px

## Иконки

- **Стиль:** Sharp line, minimal
- **Толщина:** 1.5px
- **Размер:** 18px sidebar, 20px карточки
- **Цвет:** #8A8A8A, active — #5E6AD2
- **Без контейнеров** — иконки standalone

## Графики

- **Стиль:** Stroke-only, minimal
- **Толщина:** 1.5px
- **Цвет:** #5E6AD2
- **Сетка:** Минимальная, rgba(255,255,255,0.04)
- **Без заливки** — только линии

## Глубина интерфейса

**Flat:**
- Абсолютно плоский
- Разделение через border, не тени
- Minimum visual noise
- Ultra-clean, data-focused

## Свет

- Нет glow-эффектов
- Равномерный flat light
- Минимальные тени только для вложенности

## Blur

Не используется.

## Скругления

```
Cards:           8px (маленькие)
Buttons:         6px
Icon containers: 6px
Charts:          6px
Inputs:          6px
```

## Тени

```
Card:            0 1px 2px rgba(0, 0, 0, 0.2)
Dropdown:        0 4px 16px rgba(0, 0, 0, 0.3)
```

## Анимации

- **Hover:** Border brighten, 150ms
- **Transition:** Быстрые, subtle, 100-150ms
- **Нет декоративных анимаций** — всё функционально

## UI Kit

```
Button Primary:      bg #5E6AD2, text white, radius 6px
Button Secondary:    bg rgba(94,106,210,0.12), text #5E6AD2, radius 6px
Button Ghost:        bg transparent, text #8A8A8A, hover rgba(255,255,255,0.04)
Card:                bg #1A1A1A, border rgba(255,255,255,0.06), radius 8px
Metric:              value 28px/600, label 12px/400
Progress:            bg rgba(255,255,255,0.06), fill #5E6AD2, radius 4px
Sidebar Item:        padding 8px 12px, radius 6px
Quick Action:        bg #1A1A1A, border, radius 8px
Badge:               bg rgba(94,106,210,0.12), text #5E6AD2, radius 4px
```

## Дизайн-система

- **Grid:** 12 колонок, 20px gutter (tighter)
- **Spacing scale:** 4, 8, 12, 16, 20, 24, 32
- **Radius scale:** 4, 6, 8
- **Shadow scale:** 1 уровень (minimal)
- **Typography scale:** 12, 13, 15, 18, 24, 28

## Что обязательно сохранить

> Ultra-minimalism — никаких декораций.
> Маленькие radius (8px) — key signature.
> Stroke-only графики без заливки.
> Minimum spacing,紧凑 компоновка.
> Чистая функциональность, data-first.

## Что нельзя менять

- Нельзя добавлять gradient и glow
- Нельзя делать large radius
- Нельзя добавлять decorative элементы
- Нельзя делать светлый фон
- Нельзя перегружать тенями

## Prompt для DeepSeek

> Создай Ultra-minimal dark dashboard в стиле Linear. Фон #0D0D0D, карточки #1A1A1A с тонкими border rgba(255,255,255,0.06). Акцент #5E6AD2. Sidebar 240px ultra-clean. Radius 8px, минимальные spacing. Графики stroke-only без заливки. Иконки 1.5px strokeWidth, standalone без контейнеров. Никаких gradient, glow, decoration. Чистая функциональность, data-first подход. Для productivity-oriented финтех-инструментов.

---

# Стиль №9 — Revolut Style

## Общая концепция

Чистый тёмный интерфейс в стиле Revolut. Функциональный, с акцентом на быстрые действия и обзор. Понятный и доступный для широкой аудитории. Вызывает ощущение modern banking, convenience, control. Напоминает: Revolut, Monzo, N26. Подходит для consumer banking, повседневных финтех-приложений.

## Общая композиция

- **Sidebar:** 240px, тёмный
- **Header:** Заголовок «Обзор» + period selector
- **Сетка:** 12 колонок
- **Карточки:** Сбалансированные, 2-3 колонки
- **Вертикальный rhythm:** 24px
- **Hero-блок:** Крупный баланс по центру, минимум деталей

## Цветовая палитра

```
Фон основной:        #191C1F (dark grey)
Фон sidebar:         #121517
Карточки:            #23272B
Карточки hover:      #2C3035
Primary:             #0075EB (Revolut blue)
Primary light:       rgba(0, 117, 235, 0.12)
Success:             #00C853
Danger:              #FF3B30
Warning:             #FFB800
Текст основной:      #FFFFFF
Текст вторичный:     #9CA3AF
Текст третичный:     #6B7280
Border:              rgba(255, 255, 255, 0.08)
Border hover:        rgba(255, 255, 255, 0.14)
Shadow:              0 2px 8px rgba(0, 0, 0, 0.25)
```

## Типографика

```
Шрифт:               Inter

Page title:          24px / 700 / -0.01em
Section title:       16px / 600
Card title:          14px / 500
Body:                14px / 400 / line-height 1.5
Caption:             12px / 500
Metric value:        36px / 700 / -0.02em (крупный баланс)
Metric label:        13px / 500 / text-secondary
Sidebar item:        13px / 500
```

## Sidebar

- **Ширина:** 240px
- **Фон:** #121517, border-right: 1px solid rgba(255,255,255,0.06)
- **Логотип:** «FinHelper» — 16px / 700
- **Навигация:** Иконки 20px strokeWidth 1.75, текст 13px/500
- **Active:** Фон rgba(0, 117, 235, 0.1), текст #FFFFFF, иконка #0075EB
- **Hover:** rgba(255, 255, 255, 0.04)
- **Разделители:** rgba(255,255,255,0.06)

## Header / Hero-блок

- **Hero-блок:** Крупный баланс по центру
- **Баланс:** 36px/700, белый, по центру hero
- **Label:** «Общий баланс» сверху, 14px/500, text-secondary
- **Доходы/Расходы:** Симметрично по бокам от баланса, 16px/500
- **Подпись:** «Всё под контролем» — 13px/400, text-tertiary
- **Background:** #23272B
- **Radius:** 16px
- **Border:** 1px solid rgba(255,255,255,0.08)

## Карточки

### Карточка «Баланс» (Hero)
- Полноширинная
- Padding: 28px
- Background: #23272B
- Баланс по центру: 36px/700
- Доходы/Расходы симметрично
- Radius: 16px

### Карточки «Быстрые действия»
- 4 в ряд, крупные кнопки с иконками
- Padding: 20px
- Background: #23272B
- Border: 1px solid rgba(255,255,255,0.08)
- Radius: 16px
- Иконки: 48px контейнеры (крупные)
- Hover: border brighten, translateY(-1px)

### Карточки метрик
- 2 в ряд
- Padding: 20px
- Background: #23272B
- Radius: 16px
- Значение: 28px/700
- Label: 13px/500 text-secondary

### Карточки графиков
- 2 в ряд
- Padding: 20px
- Background: #23272B
- Radius: 16px

## Иконки

- **Стиль:** Rounded, bold
- **Толщина:** 2px
- **Размер:** 20px sidebar, 28px в быстрых действиях (крупные)
- **Цвет:** #9CA3AF, active — #0075EB
- **Контейнеры быстрых действий:** 48px, скругление 14px

## Графики

- **Тип:** Area chart, Bar chart, Donut chart
- **Цвета:** #0075EB, #38BDF8, #00C853
- **Сетка:** rgba(255,255,255,0.04)
- **Стиль:** Чёткий, информативный

## Глубина интерфейса

**Elevation:**
- Карточки сsoft shadows
- Hero-блок — центральный фокус
- Чёткая визуальная иерархия
- Ощущение: controlled, trustworthy

## Свет

- Равномерный
- Минимальные shadows
- Без glow
- Functional lighting

## Blur

Не используется.

## Скругления

```
Cards:           16px
Buttons:         14px
Icon containers: 14px
Charts:          14px
Inputs:          12px
```

## Тени

```
Card:            0 2px 8px rgba(0, 0, 0, 0.25)
Card hover:      0 4px 16px rgba(0, 0, 0, 0.3)
Dropdown:        0 8px 24px rgba(0, 0, 0, 0.4)
```

## Анимации

- **Card hover:** translateY(-2px), 200ms ease-out
- **Quick action:** scale(0.97) on press, 100ms
- **Transition:** Smooth, 200ms

## UI Kit

```
Button Primary:      bg #0075EB, text white, radius 14px
Button Secondary:    bg rgba(0,117,235,0.12), text #0075EB, radius 14px
Button Ghost:        bg transparent, text #9CA3AF, hover rgba(255,255,255,0.04)
Button Icon:         48px, radius 14px
Card:                bg #23272B, border rgba(255,255,255,0.08), radius 16px
Metric:              value 28px/700, label 13px/500
Progress:            bg rgba(0,117,235,0.15), fill #0075EB, radius 999px
Sidebar Item:        padding 10px 16px, radius 10px
Quick Action:        bg #23272B, border, icon 48px, radius 16px
Badge:               bg rgba(0,117,235,0.15), text #38BDF8, radius 999px
```

## Дизайн-система

- **Grid:** 12 колонок, 24px gutter
- **Spacing:** 4, 8, 12, 16, 20, 24, 28, 32, 40
- **Radius:** 10, 12, 14, 16, 20
- **Shadow:** 2 уровня
- **Typography:** 12, 13, 14, 16, 20, 24, 28, 32, 36

## Что обязательно сохранить

> Крупный баланс по центру hero — ключевая фишка Revolut.
> Симметричная компоновка доходов/расходов.
> Крупные иконки быстрых действий (48px).
> Синий акцент #0075EB.
> Понятность и доступность для широкой аудитории.

## Что нельзя менять

- Нельзя делать слишком минималистичный — должен быть понятен всем
- Нельзя убирать крупные иконки быстрых действий
- Нельзя делать мелкую типографику
- Нельзя ставить толстые рамки

## Prompt для DeepSeek

> Создай consumer banking dashboard в стиле Revolut. Тёмный фон #191C1F, карточки #23272B. Крупный баланс 36px по центру hero-блока. Симметричные доходы/расходы по бокам. 4 быстрых действия с крупными иконками 48px. Синий акцент #0075EB. Sidebar 240px. Radius 16px для карточек, 14px для кнопок. Понятный, accessible, modern banking интерфейс.

---

# Стиль №10 — Wise Inspired

## Общая концепция

Светлый интерфейс в стиле Wise (бывший TransferWise). Сдержанный, профессиональный, с характерными зелёными акцентами. Вызывает ощущение transparency, fairness, international banking. Напоминает: Wise, N26, Monzo Light. Подходит для международных финтех-платформ, валютных операций, переводов.

## Общая композиция

- **Sidebar:** 240px, белый/cream с border справа
- **Header:** Заголовок «Обзор» + period selector
- **Сетка:** 12 колонок, просторная
- **Карточки:** Сбалансированные, 2-3 колонки
- **Вертикальный rhythm:** 24px
- **Hero-блок:** Сдержанный с зелёными акцентами

## Цветовая палитра

```
Фон основной:        #FFFFFF
Фон sidebar:         #F7F7F5 (warm off-white)
Карточки:            #FFFFFF
Карточки hover:      #FAFAF8
Primary:             #9FE870 (Wise bright green — характерный)
Primary dark:        #163300 (тёмно-зелёный для текста)
Success:             #2DC653
Danger:              #FF5A5F
Warning:             #FFC233
Текст основной:      #163300 (тёмно-зелёный — фирменный)
Текст вторичный:     #6B7B6B
Текст третичный:     #9BA89B
Border:              rgba(0, 0, 0, 0.08)
Border hover:        rgba(0, 0, 0, 0.14)
Shadow:              0 1px 3px rgba(0, 0, 0, 0.05)
Shadow medium:       0 4px 12px rgba(0, 0, 0, 0.08)
```

## Типографика

```
Шрифт:               Inter / Nunito Sans

Page title:          28px / 700 / -0.01em
Section title:       16px / 600
Card title:          14px / 500
Body:                14px / 400 / line-height 1.5
Caption:             12px / 500
Metric value:        32px / 700 / -0.01em
Metric label:        13px / 500 / text-secondary
Sidebar item:        13px / 500
```

## Sidebar

- **Ширина:** 240px
- **Фон:** #F7F7F5, border-right: 1px solid rgba(0,0,0,0.08)
- **Логотип:** «FinHelper» — 16px / 700, #163300
- **Навигация:** Иконки 20px strokeWidth 1.75, текст 13px/500
- **Active:** Левая border 3px #9FE870, фон rgba(159, 232, 112, 0.08), текст #163300
- **Hover:** rgba(0, 0, 0, 0.03)
- **Разделители:** rgba(0,0,0,0.06)

## Header / Hero-блок

- **Hero-блок:** Сдержанный, без громких gradient
- **Баланс:** 32px/700, #163300
- **Label:** «Баланс» — 14px/500, #6B7B6B
- **Доходы/Расходы:** Справа, с цветовыми индикаторами
- **Background:** #FFFFFF
- **Border:** 1px solid rgba(0,0,0,0.08)
- **Radius:** 16px

## Карточки

### Карточка «Баланс» (Hero)
- Полноширинная
- Padding: 24px
- Background: #FFFFFF
- Border: 1px solid rgba(0,0,0,0.08)
- Radius: 16px
- Shadow: 0 1px 3px rgba(0,0,0,0.05)

### Карточки «Быстрые действия»
- 4 в ряд
- Padding: 16px
- Background: #FFFFFF
- Border: 1px solid rgba(0,0,0,0.08)
- Radius: 12px
- Иконки: 40px контейнеры сsoft green background
- Hover: border rgba(0,0,0,0.14)

### Карточки метрик
- 2 в ряд
- Padding: 20px
- Background: #FFFFFF
- Border: 1px solid rgba(0,0,0,0.08)
- Radius: 12px
- Значение: 28px/700
- Label: 13px/500 #6B7B6B

### Карточки графиков
- 2 в ряд
- Padding: 20px
- Background: #FFFFFF
- Border: 1px solid rgba(0,0,0,0.08)
- Radius: 12px

## Иконки

- **Стиль:** Rounded line
- **Толщина:** 1.75px
- **Размер:** 20px sidebar, 24px карточки
- **Цвет:** #6B7B6B, active — #163300
- **Контейнеры быстрых действий:** 40px, скругление 12px, фон rgba(159, 232, 112, 0.12)

## Графики

- **Тип:** Area chart, Bar chart, Donut chart
- **Цвета:** #9FE870, #2DC653, #163300
- **Сетка:** rgba(0, 0, 0, 0.04)
- **Заливка:** rgba(159, 232, 112, 0.2)

## Глубина интерфейса

**Soft:**
- Мягкие тени
- White cards на slightly off-white фоне
- Ощущение: clean, transparent, fair

## Свет

- Рассеянный, равномерный
- Тёплый undertone
- Без glow

## Blur

Не используется.

## Скругления

```
Cards:           16px (hero), 12px (остальные)
Buttons:         12px
Icon containers: 12px
Charts:          12px
Inputs:          10px
```

## Тени

```
Card:            0 1px 3px rgba(0, 0, 0, 0.05)
Card hover:      0 4px 12px rgba(0, 0, 0, 0.08)
Dropdown:        0 8px 24px rgba(0, 0, 0, 0.12)
```

## Анимации

- **Card hover:** translateY(-1px), 200ms
- **Button press:** scale(0.98), 100ms
- **Transition:** Smooth, 200ms

## UI Kit

```
Button Primary:      bg #9FE870, text #163300, radius 12px
Button Secondary:    bg rgba(159,232,112,0.12), text #163300, radius 12px
Button Ghost:        bg transparent, text #6B7B6B, hover rgba(0,0,0,0.03)
Button Icon:         40px, radius 12px
Card:                bg white, border rgba(0,0,0,0.08), radius 12px
Metric:              value 28px/700, label 13px/500
Progress:            bg rgba(159,232,112,0.15), fill #9FE870, radius 999px
Sidebar Item:        padding 10px 16px, radius 8px, left-border active
Quick Action:        bg white, border, radius 12px, icon 40px
Badge:               bg rgba(159,232,112,0.12), text #163300, radius 999px
```

## Дизайн-система

- **Grid:** 12 колонок, 24px gutter
- **Spacing:** 4, 8, 12, 16, 20, 24, 28, 32, 40
- **Radius:** 8, 10, 12, 16
- **Shadow:** 2 уровня
- **Typography:** 12, 13, 14, 16, 20, 24, 28, 32

## Что обязательно сохранить

> Зелёный акцент #9FE870 — фирменный цвет Wise.
> Тёмно-зелёный текст #163300 вместо чёрного.
> Сдержанность и professional вид.
> Ощущение transparency и fairness.

## Что нельзя менять

- Нельзя делать тёмный фон
- Нельзя убирать зелёный акцент
- Нельзя ставить толстые рамки
- Нельзя делать агрессивные цвета

## Prompt для DeepSeek

> Создай international fintech dashboard в стиле Wise. Белый фон #FFFFFF, sidebar #F7F7F5. Карточки белые с border rgba(0,0,0,0.08). Зелёный акцент #9FE870, тёмно-зелёный текст #163300. Sidebar 240px с left-border active. Radius 12px. Сдержанный, professional, transparent. Для международных финтех-платформ и валютных операций.

---

# Стиль №11 — Glassmorphism Light

## Общая концепция

Полноценный glassmorphism на светлом фоне. Все элементы — стеклянные, полупрозрачные, с backdrop blur. Вызывает ощущение futuristic elegance, lightness, airiness. Напоминает: macOS Big Sur, iOS, modern concept UIs. Подходит для innovative fintech, premium neobanks, концептуальных приложений.

## Общая композиция

- **Sidebar:** 240px, стеклянная с blur
- **Header:** Заголовок «Обзор» + period
- **Сетка:** 12 колонок
- **Карточки:** Все стеклянные, полупрозрачные
- **Вертикальный rhythm:** 24px
- **Hero-блок:** Стеклянный с gradient blobs на фоне

## Цветовая палитра

```
Фон основной:        #E8ECF4 (soft blue-grey)
Background blobs:    #C3B1E1 (lavender), #A7C7E7 (baby blue), #B5EAD7 (mint)
Sidebar:             rgba(255, 255, 255, 0.4) + blur
Карточки:            rgba(255, 255, 255, 0.4) + blur
Primary:             #6366F1 (indigo)
Success:             #22C55E
Danger:              #EF4444
Текст основной:      #1E293B
Текст вторичный:     #64748B
Текст третичный:     #94A3B8
Border:              rgba(255, 255, 255, 0.5)
Border hover:        rgba(255, 255, 255, 0.7)
Shadow:              0 8px 32px rgba(0, 0, 0, 0.08)
```

## Типографика

```
Шрифт:               Inter

Page title:          28px / 700
Section title:       16px / 600
Card title:          14px / 500
Body:                13px / 400
Caption:             11px / 500
Metric value:        32px / 700
Sidebar item:        13px / 500
```

## Sidebar

- **Ширина:** 240px
- **Background:** rgba(255, 255, 255, 0.4)
- **Backdrop-filter:** blur(20px) saturate(180%)
- **Border:** 1px solid rgba(255,255,255,0.5)
- **Логотип:** «FinHelper» — 16px / 700
- **Active:** rgba(99, 102, 241, 0.15), текст #6366F1
- **Hover:** rgba(255, 255, 255, 0.3)

## Header / Hero-блок

- **Background blobs:** 3-4 radial gradient элемента на фоне (#C3B1E1, #A7C7E7, #B5EAD7)
- **Hero:** Стеклянная карточка поверх blobs
- **Баланс:** 32px/700, #1E293B
- **Blur:** backdrop-filter: blur(16px) saturate(160%)
- **Border:** 1px solid rgba(255,255,255,0.5)

## Карточки

### Все карточки
- **Background:** rgba(255, 255, 255, 0.4)
- **Backdrop-filter:** blur(20px) saturate(180%)
- **Border:** 1px solid rgba(255,255,255,0.5)
- **Radius:** 24px
- **Shadow:** 0 8px 32px rgba(0, 0, 0, 0.08)
- **Padding:** 24px

## Иконки

- **Стиль:** Rounded line
- **Толщина:** 1.75px
- **Размер:** 20px
- **Контейнеры:** Пастельные gradient фоны

## Графики

- **Цвета:** Мягкие пастельные
- **Заливка:** Полупрозрачные gradient'ы
- **Сетка:** Минимальная

## Глубина интерфейса

**Full Glassmorphism:**
- Фоновые gradient blobs видны сквозь карточки
- Многослойность ключевая фишка
- Ощущение: floating, ethereal, futuristic

## Свет

- Gradient blobs как light sources
- Ambient lighting через прозрачность
- Мягкие цвета

## Blur

```
Sidebar:     backdrop-filter: blur(20px) saturate(180%)
Cards:       backdrop-filter: blur(20px) saturate(180%)
Hero:        backdrop-filter: blur(16px) saturate(160%)
Dropdown:    backdrop-filter: blur(24px) saturate(200%)
```

## Скругления

```
Cards:       24px (большие)
Buttons:     16px
Icon cont.:  16px
Inputs:      14px
Charts:      20px
```

## Тени

```
Card:        0 8px 32px rgba(0, 0, 0, 0.08)
Card hover:  0 12px 40px rgba(0, 0, 0, 0.12)
```

## Анимации

- **Blobs:** Медленное движение (15-20s infinite)
- **Hover:** translateY(-2px), shadow increase, 200ms
- **Transition:** Smooth, 200-300ms

## UI Kit

```
Button Primary:      bg #6366F1, text white, radius 16px
Button Secondary:    bg rgba(255,255,255,0.5), text #1E293B, radius 16px
Card:                bg rgba(255,255,255,0.4), blur, border rgba(255,255,255,0.5), radius 24px
Progress:            bg rgba(99,102,241,0.15), fill #6366F1, radius 999px
```

## Что обязательно сохранить

> Gradient blobs на фоне — ключевая фишка.
> Все элементы стеклянные с backdrop blur.
> Большие radius (24px).
> Ощущение futures и elegance.

## Prompt для DeepSeek

> Создай Glassmorphism light dashboard. Фон #E8ECF4 с gradient blobs (лавандовый #C3B1E1, голубой #A7C7E7, мятный #B5EAD7). Все карточки rgba(255,255,255,0.4) с backdrop blur 20px saturate 180%. Border rgba(255,255,255,0.5). Radius 24px. Sidebar стеклянная. Ощущение futuristic elegance, lightness, airiness.

---

# Стиль №12 — Soft Neumorphism

## Общая концепция

Мягкий неоморфизм — карточки с выпуклыми/вогнутыми тенями, создающими ощущение 3D-поверхности. Вызывает ощущение taktility, softness, modern craft. Напоминает: neumorphism concept UIs,某些 Android designs. Подходит для innovative fintech, концептуальных приложений, premium neobanks.

## Общая композиция

- **Sidebar:** 220px, neumorphic
- **Header:** Заголовок «Обзор»
- **Сетка:** 12 колонок
- **Карточки:** Neumorphic raised/pressed
- **Вертикальный rhythm:** 24px
- **Hero-блок:** Neumorphic raised

## Цветовая палитра

```
Фон/Основа:          #E4E9F0 (base — все элементы этого цвета)
Shadow light:        #FFFFFF (светлая тень)
Shadow dark:         #A3B1C6 (тёмная тень)
Primary:             #6C63FF
Success:             #22C55E
Danger:              #EF4444
Текст основной:      #2D3748
Текст вторичный:     #718096
Border:              Нет (тени вместо border)
```

## Типографика

```
Шрифт:               Inter

Page title:          28px / 700
Card title:          14px / 600
Body:                13px / 400
Caption:             11px / 500
Metric value:        32px / 700
```

## Sidebar

- **Фон:** #E4E9F0
- **Raised effect:** box-shadow: 6px 6px 12px #A3B1C6, -6px -6px 12px #FFFFFF
- **Active:** Inset version (pressed): box-shadow: inset 4px 4px 8px #A3B1C6, inset -4px -4px 8px #FFFFFF
- **Hover:** Мягкий transition

## Hero-блок

- **Neumorphic raised:** Двойные тени
- **Баланс:** 32px/700, #2D3748
- **Background:** #E4E9F0 (идентичен фону)
- **Radius:** 20px

## Карточки

### Raised (по умолчанию)
- **Background:** #E4E9F0
- **Shadow:** box-shadow: 6px 6px 12px #A3B1C6, -6px -6px 12px #FFFFFF
- **Radius:** 20px
- **Нет border** — разделение через тени

### Pressed (для input, progress, active)
- **Background:** #E4E9F0
- **Shadow:** box-shadow: inset 4px 4px 8px #A3B1C6, inset -4px -4px 8px #FFFFFF
- **Radius:** 12px

## Иконки

- **Стиль:** Rounded, soft
- **Толщина:** 1.75px
- **Размер:** 20px
- **Контейнеры:** Neumorphic raised

## Графики

- **Стиль:** Neumorphic elements для data visualization
- **Цвета:** #6C63FF и оттенки

## Глубина интерфейса

**Neumorphism:**
- Тени создают ощущение 3D-объёма
- Raised vs Pressed состояния
- Taktil feel — хочется нажать
- Ощущение: soft, modern, tactile

## Свет

- Два источника теней (светлая + тёмная)
- Мягкий, рассеянный
- Создаёт 3D-эффект

## Blur

Не используется. Глубина через тени.

## Скругления

```
Cards:       20px
Buttons:     14px
Icon cont.:  14px
Inputs:      12px (pressed state)
Charts:      16px
```

## Тени

```
Raised:      6px 6px 12px #A3B1C6, -6px -6px 12px #FFFFFF
Pressed:     inset 4px 4px 8px #A3B1C6, inset -4px -4px 8px #FFFFFF
Hover:       8px 8px 16px #A3B1C6, -8px -8px 16px #FFFFFF
```

## Анимации

- **Hover:** Усиление теней (8px вместо 6px), 200ms
- **Press:** Transition от raised к pressed, 150ms
- **Нет translateY** — глубина через тени

## UI Kit

```
Button Raised:    bg #E4E9F0, shadow raised, radius 14px
Button Pressed:   bg #E4E9F0, shadow pressed, radius 14px
Card:             bg #E4E9F0, shadow raised, radius 20px
Input:            bg #E4E9F0, shadow pressed, radius 12px
Progress:         bg #E4E9F0 (pressed), fill #6C63FF (raised), radius 999px
Toggle:           raised/pressed states
```

## Что обязательно сохранить

> Neumorphic shadows — ключевая фишка.
> Raised vs Pressed состояния.
> Taktil feel — ощущение 3D-поверхности.
> Единый цвет #E4E9F0 для всех элементов.

## Prompt для DeepSeek

> Создай Neumorphism dashboard. Фон #E4E9F0, все элементы этого же цвета. Карточки с dual shadows: 6px 6px 12px #A3B1C6, -6px -6px 12px #FFFFFF. Pressed states: inset shadows. Radius 20px. Taktil, soft, modern. Primary #6C63FF. Ощущение 3D-поверхности и тактильности.

---

# Стиль №13 — Gradient Hero

## Общая концепция

Интерфейс с доминирующим gradient hero-блоком. Яркий градиент привлекает внимание к балансу, остальная часть — сдержанная. Вызывает ощущение freshness, growth, positivity. Напоминает: many modern fintech hero sections. Подходит для приложений по накоплениям, инвестированию, экологичным финансам.

## Общая композиция

- **Sidebar:** 240px, белый
- **Header:** Заголовок «Обзор» + period
- **Сетка:** 12 колонок
- **Карточки:** 2-3 колонки
- **Вертикальный rhythm:** 24px
- **Hero-блок:** Крупный gradient — центральный элемент

## Цветовая палитра

```
Фон основной:        #F1F5F9 (cool grey)
Sidebar:             #FFFFFF
Hero gradient:       #059669 → #10B981 → #34D399 (green gradient)
Карточки:            #FFFFFF
Primary:             #059669 (emerald)
Success:             #22C55E
Danger:              #EF4444
Текст основной:      #0F172A
Текст вторичный:     #64748B
Текст третичный:     #94A3B8
Border:              rgba(0, 0, 0, 0.06)
Shadow:              0 1px 3px rgba(0, 0, 0, 0.06)
```

## Типографика

```
Шрифт:               Inter

Page title:          28px / 700
Card title:          14px / 500
Body:                13px / 400
Caption:             11px / 500
Metric value (hero): 36px / 700 / white
Metric value:        28px / 700
```

## Sidebar

- **Ширина:** 240px
- **Фон:** #FFFFFF, border-right: 1px solid rgba(0,0,0,0.06)
- **Active:** Левая border 3px #059669, фон rgba(5, 150, 105, 0.06)
- **Иконки:** #64748B, active — #059669

## Hero-блок

- **Background:** Multi-stop gradient #059669 → #10B981 → #34D399
- **Баланс:** 36px/700, белый, по центру
- **Label:** «Общий баланс» — 14px/500, rgba(255,255,255,0.8)
- **Доходы/Расходы:** Справа, белые
- **Декоративные элементы:** Мягкие circles/blob на фоне gradient
- **Radius:** 24px
- **Shadow:** 0 12px 40px rgba(5, 150, 105, 0.25)

## Карточки

### Карточки «Быстрые действия»
- 4 в ряд
- Padding: 16px
- Background: #FFFFFF
- Border: 1px solid rgba(0,0,0,0.06)
- Radius: 16px
- Иконки: 40px контейнеры сsoft green background

### Карточки метрик
- 2 в ряд
- Padding: 20px
- Background: #FFFFFF
- Border: 1px solid rgba(0,0,0,0.06)
- Radius: 16px
- Значение: 28px/700
- Label: 12px/500 #64748B

### Карточки графиков
- 2 в ряд
- Background: #FFFFFF
- Border: 1px solid rgba(0,0,0,0.06)
- Radius: 16px

## Иконки

- **Стиль:** Rounded line
- **Толщина:** 1.75px
- **Размер:** 20px
- **Контейнеры быстрых действий:** 40px, фон rgba(5,150,105,0.08)

## Графики

- **Цвета:** #059669, #10B981, #34D399
- **Заливка:** rgba(5, 150, 105, 0.15)

## Глубина

**Soft + Gradient Hero:**
- Hero — яркий gradient, main focal point
- Карточки — white, soft shadows
- Fresh, growth-oriented

## Свет

- Hero gradient создаёт focal glow
- Soft shadows для остальных карточек

## Blur

Возможен轻微 на hero.

## Скругления

```
Cards:       16px
Hero:        24px
Buttons:     14px
Charts:      14px
```

## Тени

```
Card:        0 1px 3px rgba(0, 0, 0, 0.06)
Hero:        0 12px 40px rgba(5, 150, 105, 0.25)
```

## UI Kit

```
Button Primary:      bg #059669, text white, radius 14px
Button Secondary:    bg rgba(5,150,105,0.08), text #059669, radius 14px
Card:                bg white, border, radius 16px
Hero:                gradient, radius 24px
```

## Что обязательно сохранить

> Gradient hero — центральный элемент.
> Зелёный gradient #059669 → #34D399.
> Белый баланс на gradient фоне.
> Fresh, growth, positivity.

## Prompt для DeepSeek

> Создай dashboard с Gradient Hero. Hero-блок с multi-stop gradient #059669 → #34D399, баланс белый 36px по центру. Остальные карточки белые на #F1F5F9 фоне. Зелёные акценты. Fresh, modern, growth-oriented. Для приложений по накоплениям и инвестированию.

---

# Стиль №14 — Minimal Monochrome

## Общая концепция

Абсолютный минимализм в монохромной палитре. Только чёрный, белый и оттенки серого. Вызывает ощущение purity, clarity, timeless elegance. Напоминает: Apple design, Dieter Rams, Braun. Подходит для ultra-premium приложений, ценителей минимализма, дизайнеров.

## Общая композиция

- **Sidebar:** 240px, белый
- **Header:** Заголовок «Обзор»
- **Сетка:** 12 колонок, огромные отступы
- **Карточки:** Минимум элементов
- **Вертикальный rhythm:** 32px (увеличенный)
- **Hero-блок:** Типографический — баланс как текст

## Цветовая палитра

```
Фон:                 #FFFFFF
Sidebar:             #FFFFFF
Карточки:            #FFFFFF
Primary:             #000000
Text:                #000000
Text secondary:      #71717A
Text tertiary:       #A1A1AA
Border:              #E4E4E7
Shadow:              0 1px 2px rgba(0, 0, 0, 0.05)
```

## Типографика

```
Шрифт:               Inter (light/regular weights — key feature)

Page title:          28px / 300 (light weight!)
Section title:       16px / 400
Card title:          14px / 400
Body:                14px / 400
Caption:             12px / 400
Metric value:        36px / 300 (light!)
Metric label:        13px / 400
```

## Sidebar

- **Фон:** #FFFFFF, border-right: 1px solid #E4E4E7
- **Навигация:** Иконки 18px strokeWidth 1.5, текст 13px/400
- **Active:** Текст #000000, bold
- **Hover:** rgba(0, 0, 0, 0.03)

## Hero-блок

- **Нет traditional hero** — типографический подход
- **Баланс:** 36px/300, чёрный, огромный white space вокруг
- **Label:** «Баланс» — 13px/400, #71717A
- **Нет background, border, decoration** — только текст

## Карточки

### Все карточки
- **Background:** #FFFFFF
- **Border:** 1px solid #E4E4E7
- **Radius:** 12px
- **Padding:** 28-32px (generous)
- **Shadow:** 0 1px 2px rgba(0,0,0,0.05)
- **Минимум элементов** — типографика主导

## Иконки

- **Стиль:** Minimal line
- **Толщина:** 1.5px
- **Размер:** 18px
- **Цвет:** #71717A, active — #000000

## Графики

- **Стиль:** Monochrome — оттенки серого
- **Минимум** — только essential data

## Глубина

**Ultra Flat:**
- Минимум теней
- Border для разделения
- White space как design element
- Ощущение: timeless, pure, elegant

## Свет

- Равномерный
- Без directionality
- Pure white

## Blur

Не используется.

## Скругления

```
Cards:       12px
Buttons:     8px
Charts:      8px
Inputs:      6px
```

## Тени

```
Card:        0 1px 2px rgba(0, 0, 0, 0.05)
Dropdown:    0 4px 12px rgba(0, 0, 0, 0.08)
```

## Анимации

- **Минимум анимаций**
- Hover: opacity change, 150ms
- Transition: subtle, 150ms

## UI Kit

```
Button Primary:      bg #000000, text white, radius 8px
Button Secondary:    bg transparent, border #000000, text #000000, radius 8px
Button Ghost:        bg transparent, text #71717A
Card:                bg white, border #E4E4E7, radius 12px
Metric:              value 36px/300, label 13px/400
Progress:            bg #E4E4E7, fill #000000, radius 999px
```

## Что обязательно сохранить

> Light weight typography (300) — key signature.
> Огромные white space.
> Только чёрный/белый/серый.
> Timeless, pure, elegant.

## Prompt для DeepSeek

> Создай Ultra-minimal monochrome dashboard. Белый фон, чёрный текст, серые border #E4E4E7. Типографика light weight (300) для метрик и заголовков. Огромные white space, generous padding. Radius 12px, minimum shadows. Только оттенки чёрного и белого. Maximum minimalism, timeless elegance.

---

# Стиль №15 — Compact Dashboard

## Общая концепция

Плотный информативный дашборд. Максимум данных на экране, компактная компоновка, никаких пустых пространств. Вызывает ощущение productivity, information density, control. Напоминает: Bloomberg Terminal, trading platforms, admin panels. Подходит для power users, аналитиков, трейдеров, профессионалов.

## Общая композиция

- **Sidebar:** 200px (уменьшенная)
- **Header:** Заголовок + actions, компактный
- **Сетка:** 3-4 колонки, плотная сетка
- **Карточки:** Компактные, маленькие padding
- **Вертикальный rhythm:** 12-16px
- **Hero-блок:** Нет large hero — метрики в строку

## Цветовая палитра

```
Фон основной:        #F8FAFC
Sidebar:             #F1F5F9
Карточки:            #FFFFFF
Primary:             #2563EB
Success:             #16A34A
Danger:              #DC2626
Текст основной:      #0F172A
Текст вторичный:     #64748B
Текст третичный:     #94A3B8
Border:              #E2E8F0
Shadow:              0 1px 2px rgba(0, 0, 0, 0.04)
```

## Типографика

```
Шрифт:               Inter

Page title:          20px / 600 (уменьшенный)
Section title:       13px / 600
Card title:          12px / 500
Body:                12px / 400 (компактный)
Caption:             11px / 400
Metric value:        22px / 700 (компактный)
Metric label:        11px / 400
```

## Sidebar

- **Ширина:** 200px (уменьшенная)
- **Фон:** #F1F5F9
- **Навигация:** Иконки 16px strokeWidth 1.5, текст 12px/400
- **Active:** Left border 2px #2563EB
- **Компактная компоновка:** padding 6px 10px

## Hero-блок

- **Нет traditional hero**
- **Метрики в строку:** Баланс, Доходы, Расходы — все в одной горизонтальной карточке
- **Компактная layout:** padding 12-16px

## Карточки

### Карточки метрик (горизонтальные)
- **Layout:** 3-4 метрики в строку
- **Padding:** 12-16px
- **Background:** #FFFFFF
- **Border:** 1px solid #E2E8F0
- **Radius:** 8px
- **Значение:** 22px/700
- **Label:** 11px/400

### Карточки «Быстрые действия»
- 4 в ряд, компактные
- Padding: 10px
- Иконки: 28px
- Radius: 6px

### Карточки графиков
- 2-3 в ряд
- Padding: 12px
- Radius: 8px
- Компактные chart areas

## Иконки

- **Стиль:** Compact line
- **Толщина:** 1.5px
- **Размер:** 16px
- **Контейнеры:** 28px

## Графики

- **Стиль:** Dense, information-rich
- **Минимум padding** around charts
- **Small multiples** — много маленьких графиков

## Глубина

**Flat + Compact:**
- Минимум whitespace
- Border для разделения
- Density как design principle

## Свет

- Равномерный
- Functional

## Blur

Не используется.

## Скругления

```
Cards:       8px
Buttons:     6px
Icon cont.:  6px
Charts:      6px
Inputs:      4px
```

## Тени

```
Card:        0 1px 2px rgba(0, 0, 0, 0.04)
```

## Анимации

- **Минимум** — всё функционально
- Hover: subtle highlight, 100ms

## UI Kit

```
Button Primary:      bg #2563EB, text white, radius 6px
Button Secondary:    bg rgba(37,99,235,0.08), text #2563EB, radius 6px
Card:                bg white, border #E2E8F0, radius 8px
Metric:              value 22px/700, label 11px/400
Progress:            bg #E2E8F0, fill #2563EB, radius 4px
```

## Что обязательно сохранить

> Information density — ключевой принцип.
> Компактная компоновка, minimum whitespace.
> Уменьшенная типографика и spacing.
> 3-4 колонки, много данных на экране.

## Prompt для DeepSeek

> Создай Compact dashboard для power users. Фон #F8FAFC, карточки белые плотные. 3-4 колонки, метрики в строку. Padding 12-16px, radius 8px. Уменьшенная типографика 12-22px. Sidebar 200px. Maximum information density. Для аналитиков, трейдеров, профессионалов.

---

# Стиль №16 — Sakura Bloom

## Общая концепция

Романтичный, нежный интерфейс в японском стиле с темой цветущей сакуры. Доминируют оттенки розового, пудрового и белого. Стиль напоминает эстетику kawaii, японские приложения и весенние коллекции. Вызывает эмоции нежности, свежести, обновления и легкости. Подходит для lifestyle-финтеха, приложений ориентированных на женскую аудиторию, или сезонных промо-кампаний. Напоминает дизайн-язык японских банковских приложений (Sony Bank) в весенней теме.

## Общая композиция

- **Sidebar:** 240px, белый/cream сsoft pink undertone
- **Header:** «Доброе утро!» + period selector + иконки уведомлений
- **Сетка:** 12 колонок, просторная компоновка
- **Карточки:** 2 колонки для метрик, 4 для быстрых действий
- **Вертикальный rhythm:** 24px
- **Hero-блок:** Полноширинная карточка с иллюстрацией цветущей сакуры справа
- **Дополнительно:** Секции «Активные цели» и «Последние операции» справа от основного контента (2-колоночная layout)

## Цветовая палитра

```
Фон основной:        #FFF5F7 (soft pink white)
Фон sidebar:         #FFFFFF
Карточки:            #FFFFFF
Hero gradient:       #FFF0F3 → #FFE4EC (мягкий розовый)
Primary:             #F472B6 (cherry blossom pink)
Primary light:       rgba(244, 114, 182, 0.08)
Secondary:           #EC4899 (deep pink)
Success:             #22C55E
Danger:              #EF4444
Текст основной:      #1F2937
Текст вторичный:     #6B7280
Текст третичный:     #9CA3AF
Border:              rgba(244, 114, 182, 0.12)
Border hover:        rgba(244, 114, 182, 0.2)
Shadow:              0 2px 12px rgba(244, 114, 182, 0.08)
Accent (предупреждения): #F59E0B
```

## Типографика

```
Шрифт:               Inter / Nunito

Greeting:            20px / 600 («Доброе утро!» с эмодзи 🌸)
Page title:          28px / 700
Section title:       16px / 600
Card title:          14px / 500
Body:                13px / 400 / line-height 1.5
Caption:             11px / 500
Metric value:        32px / 700
Metric label:        12px / 500
Sidebar item:        13px / 500
Progress label:      11px / 400 («Цель: 20%»)
```

## Sidebar

- **Ширина:** 240px
- **Фон:** #FFFFFF, border-right: 1px solid rgba(244,114,182,0.08)
- **Логотип:** «FinHelper» — 16px / 700, #EC4899
- **Навигация:** Иконки 20px strokeWidth 1.75, текст 13px/500
- **Active:** Левая border 2px #F472B6, фон rgba(244,114,182,0.06), текст #F472B6
- **Hover:** rgba(244, 114, 182, 0.04)
- **Нижний элемент:** «Скрыть суммы» toggle с иконкой глаза

## Header / Hero-блок

- **Приветствие:** «Доброе утро! 🌸» — 20px/600, warm & friendly
- **Hero-блок:** Полноширинная карточка
  - Background: gradient от #FFF0F3 к #FFE4EC
  - Иллюстрация: Ветка цветущей сакуры справа (PNG/SVG, opacity 0.6-0.8), розовые лепестки, лёгкая тень на фоне
  - Баланс: «0,00 ₽» — 32px/700, #1F2937
  - Label: «Общий баланс» — 14px/500, #6B7280
  - Доходы/Расходы: Справа в hero, с цветовыми индикаторами
  - Подпись: «0,00 ₽ за месяц» — 12px/400, #9CA3AF
  - Radius: 20px
  - Border: 1px solid rgba(244,114,182,0.12)
  - Shadow: 0 8px 32px rgba(244,114,182,0.15)

## Карточки

### Карточка «Баланс» (Hero)
- Полноширинная
- Padding: 28px
- Background: gradient pink
- Иллюстрация сакуры справа
- Radius: 20px

### Карточки «Быстрые действия»
- 4 в ряд
- Padding: 16px
- Background: #FFFFFF
- Border: 1px solid rgba(244,114,182,0.1)
- Radius: 14px
- Иконки: 40px контейнеры сsoft pink background
- Hover: border rgba(244,114,182,0.2), translateY(-1px)

### Карточки «Норма сбережений» / «Изменение за месяц»
- 2 в ряд
- Padding: 20px
- Background: #FFFFFF
- Radius: 14px
- Прогресс-бар: pink gradient fill

### Секция «Обзор финансов»
- **Бюджет по категориям:** Bar chart с pink тонах
- **Расходы по категориям:** Donut chart с pink тонах

### Секции справа (Desktop)
- **Активные цели:** Список, «Нет активных целей» + «Добавить первую цель»
- **Последние операции:** Список, «Нет операций» + «Добавить первую операцию»
- Padding: 16px, Radius: 14px

## Иконки

- **Стиль:** Rounded, мягкие
- **Толщина:** 1.75px
- **Размер:** 20px sidebar, 24px карточки
- **Цвет:** #6B7280, active — #F472B6
- **Контейнеры быстрых действий:** 40px, фон rgba(244,114,182,0.08)

## Графики

- **Тип:** Bar chart (бюджет), Donut chart (расходы)
- **Цвета:** #F472B6, #EC4899, #FBCFE8, #FCE7F3 (pink palette)
- **Сетка:** rgba(244,114,182,0.06)

## Глубина интерфейса

**Soft + Illustrated:**
- Иллюстрации сакуры — главный визуальный акцент
- Мягкие тени с pink undertone
- Ощущение: spring, freshness, gentle beauty

## Свет

- Тёплый, spring-like
- Розоватый undertone в тенях
- Мягкий, рассеянный

## Blur

Не используется. Иллюстрации — декоративный элемент.

## Скругления

```
Cards:           20px (hero), 14px (остальные)
Buttons:         14px
Icon containers: 14px
Charts:          14px
Inputs:          12px
```

## Тени

```
Card:            0 2px 12px rgba(244, 114, 182, 0.08)
Card hover:      0 6px 20px rgba(244, 114, 182, 0.12)
Dropdown:        0 8px 24px rgba(0, 0, 0, 0.08)
```

## Анимации

- **Лепестки:** Возможна subtle falling petals animation (CSS, опционально)
- **Card hover:** translateY(-2px), shadow increase, 200ms
- **Progress bar:** Animated fill, 600ms ease-out

## UI Kit

```
Button Primary:      bg #F472B6, text white, radius 14px
Button Secondary:    bg rgba(244,114,182,0.08), text #F472B6, radius 14px
Button Ghost:        bg transparent, text #6B7280
Card:                bg white, border rgba(244,114,182,0.1), radius 14px
Hero:                bg gradient pink, radius 20px
Metric:              value 28px/700, label 12px/500
Progress:            bg rgba(244,114,182,0.1), fill #F472B6, radius 999px
Sidebar Item:        padding 10px 16px, radius 10px, left-border active
Quick Action:        bg white, border, radius 14px, icon 40px
Right Panel Cards:   bg white, border, radius 14px
```

## Дизайн-система

- **Grid:** 12 колонок, 24px gutter, additional right sidebar for goals/operations
- **Spacing:** 4, 8, 12, 16, 20, 24, 28, 32
- **Radius:** 10, 12, 14, 16, 20
- **Shadow:** 2 уровня (sm, md) с pink undertone
- **Typography:** 11, 12, 13, 14, 16, 20, 24, 28, 32

## Что обязательно сохранить

> Иллюстрация цветущей сакуры в hero — ключевая фишка.
> Розовая палитра от #F472B6 к #FCE7F3.
> Приветствие «Доброе утро!» с эмодзи.
> Правая панель с «Активные цели» и «Последние операции».
> Ощущение свежести, весны, обновления.

## Что нельзя менять

- Нельзя убирать иллюстрацию сакуры
- Нельзя делать холодные оттенки
- Нельзя ставить тяжёлые рамки
- Нельзя делать тёмный фон
- Нельзя убирать приветствие

## Prompt для DeepSeek

> Создай весенний dashboard финансового приложения в стиле Sakura Bloom. Фон #FFF5F7, карточки белые с pink border rgba(244,114,182,0.1). Hero-блок с gradient #FFF0F3 → #FFE4EC и иллюстрацией цветущей сакуры справа. Приветствие «Доброе утро! 🌸». Баланс 32px. Pink акцент #F472B6. Sidebar 240px белый. Правая панель с «Активные цели» и «Последние операции». Radius 14px для карточек, 20px для hero. Мягкие тени с pink undertone. Ощущение свежести, весны, женственности.

---

# Стиль №17 — Sunrise Warmth

## Общая концепция

Тёплый, уютный интерфейс с темой рассвета и горного пейзажа. Доминируют оттенки оранжевого, желтого, персикового и теплого коричневого. Стиль напоминает утреннюю свежесть, начало нового дня, оптимизм и энергию. Подходит для приложений, которые хотят создать ощущение мотивации, позитива и движения вперед. Напоминает дизайн travel-приложений и wellness-платформ с утренними ритуалами.

## Общая композиция

- **Sidebar:** 240px, белый с warm undertone
- **Header:** «Доброе утро! ☀️»
- **Сетка:** 12 колонок
- **Hero:** Иллюстрация рассвета солнца

## Цветовая палитра

```
Фон основной:        #FFFBF5 (warm white)
Sidebar:             #FFFFFF
Hero gradient:       #FEF3C7 → #FDE68A (golden)
Primary:             #F59E0B (amber/sunrise)
Primary light:       rgba(245, 158, 11, 0.08)
Secondary:           #D97706
Success:             #22C55E
Danger:              #EF4444
Текст основной:      #1F2937
Текст вторичный:     #6B7280
Текст третичный:     #9CA3AF
Border:              rgba(245, 158, 11, 0.12)
Shadow:              0 2px 12px rgba(245, 158, 11, 0.08)
```

## Типографика

```
Шрифт:               Inter / Nunito

Greeting:            20px / 600 («Доброе утро! ☀️»)
Page title:          28px / 700
Card title:          14px / 500
Body:                13px / 400
Metric value:        32px / 700
```

## Hero-блок

- **Background:** Gradient golden #FEF3C7 → #FDE68A
- **Иллюстрация:** Горы с восходящим солнцем, облака, тёплые лучи (справа, opacity 0.6-0.8)
- **Баланс:** 32px/700, тёмный
- **Label:** «Общий баланс» — 14px/500, #6B7280
- **Доходы/Расходы:** Справа в hero
- **Radius:** 20px
- **Border:** 1px solid rgba(245,158,11,0.15)
- **Shadow:** 0 8px 32px rgba(245,158,11,0.12)

## Карточки

- **Фон:** #FFFFFF
- **Border:** rgba(245,158,11,0.1)
- **Radius:** 14px
- **Иконки контейнеры:** 40px, golden background

## Графики

- **Цвета:** #F59E0B, #FBBF24, #FDE68A (amber palette)

## Глубина

**Soft + Illustrated:**
- Иллюстрация рассвета — focal point
- Warm shadows
- Optimistic, energetic

## Prompt для DeepSeek

> Создай тёплый dashboard в стиле Sunrise Warmth. Фон #FFFBF5, hero сgradient #FEF3C7 → #FDE68A и иллюстрацией рассвета. Приветствие «Доброе утро! ☀️». Amber акцент #F59E0B. Карточки белые с warm border. Radius 14px. Ощущение тепла, оптимизма, энергии.

---

# Стиль №18 — Ocean Breeze

## Общая концепция

Свежий, воздушный морской стиль с иллюстрацией океана и парусника. Доминируют оттенки голубого, бирюзового, небесного и белого. Вызывает ощущение морского бриза, свободы, путешествий и безмятежности. Подходит для приложений, связанных с путешествиями, отдыхом, или для финтеха, который хочет создать ощущение легкости и отсутствия границ.

## Общая композиция

- **Sidebar:** 240px, белый с blue undertone
- **Header:** «Доброе утро! 🐬»
- **Сетка:** 12 колонок
- **Hero:** Иллюстрация океана с парусником

## Цветовая палитра

```
Фон основной:        #F0F9FF (sky blue white)
Sidebar:             #FFFFFF
Hero gradient:       #DBEAFE → #BFDBFE (ocean blue)
Primary:             #3B82F6 (ocean blue)
Primary light:       rgba(59, 130, 246, 0.08)
Secondary:           #06B6D4 (teal)
Success:             #22C55E
Danger:              #EF4444
Текст основной:      #1E3A5F
Текст вторичный:     #64748B
Border:              rgba(59, 130, 246, 0.12)
Shadow:              0 2px 12px rgba(59, 130, 246, 0.08)
```

## Hero-блок

- **Background:** Gradient ocean blue #DBEAFE → #BFDBFE
- **Иллюстрация:** Океан, парусник, чайки в небе, лёгкие волны (справа, opacity 0.6-0.8)
- **Баланс:** 32px/700, #1E3A5F
- **Label:** «Общий баланс» — 14px/500, #64748B
- **Доходы/Расходы:** Справа в hero, с цветовыми индикаторами
- **Radius:** 20px
- **Border:** 1px solid rgba(59,130,246,0.15)
- **Shadow:** 0 8px 32px rgba(59,130,246,0.12)

## Карточки

- **Фон:** #FFFFFF
- **Border:** rgba(59,130,246,0.1)
- **Radius:** 14px

## Графики

- **Цвета:** #3B82F6, #60A5FA, #93C5FD, #06B6D4

## Prompt для DeepSeek

> Создай морской dashboard в стиле Ocean Breeze. Фон #F0F9FF, hero с gradient #DBEAFE → #BFDBFE и иллюстрацией океана с парусником. Приветствие «Доброе утро! 🐬». Blue акцент #3B82F6. Свежий, spacious, calm. Radius 14px.

---

# Стиль №19 — Lavender Dream

## Общая концепция

Мечтательный, успокаивающий интерфейс с темой лавандовых полей. Доминируют оттенки фиолетового, сиреневого, лавандового и нежно-розового. Стиль напоминает прованские пейзажи, релаксацию, медитацию и спокойствие. Подходит для wellness-финтеха, приложений ориентированных на осознанность, гармонию и баланс. Напоминает дизайн-язык приложений для медитации (Calm, Headspace) в сочетании с финансовым трекером.

## Общая композиция

- **Sidebar:** 240px, белый с lavender undertone
- **Header:** «Доброе утро! 🌿»
- **Hero:** Иллюстрация лавандовых полей

## Цветовая палитра

```
Фон основной:        #FAF5FF (lavender white)
Sidebar:             #FFFFFF
Hero gradient:       #EDE9FE → #DDD6FE (lavender)
Primary:             #8B5CF6 (violet/lavender)
Primary light:       rgba(139, 92, 246, 0.08)
Success:             #22C55E
Danger:              #EF4444
Текст основной:      #1E1B4B
Текст вторичный:     #6B7280
Border:              rgba(139, 92, 246, 0.12)
Shadow:              0 2px 12px rgba(139, 92, 246, 0.08)
```

## Hero-блок

- **Background:** Gradient lavender #EDE9FE → #DDD6FE
- **Иллюстрация:** Лавандовые поля, холмы, горы на горизонте, нежно-фиолетовое небо (справа)
- **Баланс:** 32px/700, #1E1B4B
- **Label:** «Общий баланс» — 14px/500, #6B7280
- **Radius:** 20px
- **Border:** 1px solid rgba(139,92,246,0.15)
- **Shadow:** 0 8px 32px rgba(139,92,246,0.12)
- **Radius:** 20px

## Карточки

- **Фон:** #FFFFFF
- **Border:** rgba(139,92,246,0.1)
- **Radius:** 14px

## Графики

- **Цвета:** #8B5CF6, #A78BFA, #C4B5FD (violet palette)

## Prompt для DeepSeek

> Создай лавандовый dashboard в стиле Lavender Dream. Фон #FAF5FF, hero с gradient #EDE9FE → #DDD6FE и иллюстрацией лавандовых полей. Violet акцент #8B5CF6. Спокойный, гармоничный, elegant. Radius 14px.

---

# Стиль №20 — Mint Fresh

## Общая концепция

Кристально чистый, экологичный интерфейс с акцентом на свежесть и рост. Доминируют оттенки мяты, светло-зеленого и белого. Стиль напоминает приложения для здорового образа жизни, эко-финтеха или сервисов для инвестирования в «зеленые» облигации. Вызывает эмоции чистоты, стабильности, роста и гармонии с природой. Идеально для приложений, которые хотят подчеркнуть прозрачность операций и «здоровье» финансов пользователя.

## Общая композиция

- **Sidebar:** 240px, белый с green undertone
- **Header:** «Доброе утро! 🌿»
- **Hero:** Ботанические иллюстрации листьев

## Цветовая палитра

```
Фон основной:        #F0FDF4 (mint white)
Sidebar:             #FFFFFF
Hero gradient:       #DCFCE7 → #BBF7D0 (mint)
Primary:             #22C55E (green)
Primary light:       rgba(34, 197, 94, 0.08)
Secondary:           #16A34A
Success:             #22C55E
Danger:              #EF4444
Текст основной:      #14532D
Текст вторичный:     #6B7280
Border:              rgba(34, 197, 94, 0.12)
Shadow:              0 2px 12px rgba(34, 197, 94, 0.08)
```

## Hero-блок

- **Background:** Gradient mint #DCFCE7 → #BBF7D0
- **Иллюстрации:** Ветки со свежими зелёными листьями (мята, эвкалипт) по углам hero, капли росы
- **Баланс:** 32px/700, #14532D
- **Label:** «Общий баланс» — 14px/500, #6B7280
- **Radius:** 20px
- **Border:** 1px solid rgba(34,197,94,0.15)
- **Shadow:** 0 8px 32px rgba(34,197,94,0.10)

## Карточки

- **Фон:** #FFFFFF
- **Border:** rgba(34,197,94,0.1)
- **Radius:** 14px

## Графики

- **Цвета:** #22C55E, #4ADE80, #86EFAC (green palette)

## Prompt для DeepSeek

> Создай мятный dashboard в стиле Mint Fresh. Фон #F0FDF4, hero сgradient #DCFCE7 → #BBF7D0 и ботаническими иллюстрациями листьев. Green акцент #22C55E. Свежий, чистый, growth-oriented. Radius 14px.

---

# Стиль №21 — Coral Reef

## Общая концепция

Яркий, погружающий подводный стиль с иллюстрацией кораллового рифа и рыбок. Доминируют оттенки бирюзового, циана, глубокого синего и кораллового (акцент). Напоминает исследование океана, глубину, спокойствие и скрытые сокровища. Подходит для приложений, связанных с путешествиями, дайвингом, или для финтеха, который хочет выделиться необычной, но гармоничной палитрой.

## Общая композиция

- **Sidebar:** 240px, белый с coral undertone
- **Header:** «Доброе утро! 🐠»
- **Hero:** Подводная иллюстрация с кораллами и рыбками

## Цветовая палитра

```
Фон основной:        #FFF7ED (coral white)
Sidebar:             #FFFFFF
Hero gradient:       #FFEDD5 → #FED7AA (coral)
Primary:             #F97316 (coral orange)
Primary light:       rgba(249, 115, 22, 0.08)
Secondary:           #FB923C
Accent:              #06B6D4 (cyan для воды)
Success:             #22C55E
Danger:              #EF4444
Текст основной:      #1C1917
Текст вторичный:     #6B7280
Border:              rgba(249, 115, 22, 0.12)
Shadow:              0 2px 12px rgba(249, 115, 22, 0.08)
```

## Hero-блок

- **Background:** Gradient coral #FFEDD5 → #FED7AA
- **Иллюстрация:** Коралловый риф, рыбки, пузырьки воды
- **Баланс:** 32px/700
- **Radius:** 20px

## Карточки

- **Фон:** #FFFFFF
- **Border:** rgba(249,115,22,0.1)
- **Radius:** 14px

## Графики

- **Цвета:** #F97316, #FB923C, #FDBA74, #06B6D4

## Глубина

**Illustrated + Playful:**
- Подводные иллюстрации — main feature
- Яркие,energetic colors
- Playful, fun atmosphere

## Prompt для DeepSeek

> Создай подводный dashboard в стиле Coral Reef. Фон #FFF7ED, hero сgradient #FFEDD5 → #FED7AA и иллюстрацией кораллового рифа с рыбками. Coral акцент #F97316. Игривый, энергичный, tropical. Radius 14px.

---

# Стиль №22 — Autumn Leaves

## Общая концепция

Тёплый, уютный и слегка ностальгический интерфейс, вдохновленный красотой осеннего леса. Доминируют оттенки янтарного, оранжевого, коричневого и теплого бежевого. Стиль напоминает уютные вечера с пледом, прогулки по парку и сбор урожая. Подходит для приложений, которые хотят создать ощущение комфорта, стабильности и «теплого» отношения к пользователю. Идеально для сезонных тем.

## Общая композиция

- **Sidebar:** 240px, белый с warm undertone
- **Header:** «Доброе утро! 🍂»
- **Hero:** Осенняя иллюстрация с кленовыми листьями

## Цветовая палитра

```
Фон основной:        #FFFBEB (autumn white)
Sidebar:             #FFFFFF
Hero gradient:       #FEF3C7 → #FDE68A (golden autumn)
Primary:             #D97706 (amber)
Primary light:       rgba(217, 119, 6, 0.08)
Secondary:           #B45309
Accent:              #DC2626 (red maple leaf)
Success:             #22C55E
Danger:              #EF4444
Текст основной:      #1C1917
Текст вторичный:     #6B7280
Border:              rgba(217, 119, 6, 0.12)
Shadow:              0 2px 12px rgba(217, 119, 6, 0.08)
```

## Hero-блок

- **Background:** Golden autumn gradient #FEF3C7 → #FDE68A
- **Иллюстрация:** Опавшие кленовые листья, деревья
- **Баланс:** 32px/700
- **Radius:** 20px

## Карточки

- **Фон:** #FFFFFF
- **Border:** rgba(217,119,6,0.1)
- **Radius:** 14px

## Графики

- **Цвета:** #D97706, #F59E0B, #B45309, #DC2626 (autumn palette)

## Prompt для DeepSeek

> Создай осенний dashboard в стиле Autumn Leaves. Фон #FFFBEB, hero с gradient #FEF3C7 → #FDE68A и иллюстрацией опавших кленовых листьев. Amber акцент #D97706. Тёплый, уютный, comfort. Radius 14px.

---

# Стиль №23 — Aurora Night

## Общая концепция

Магический, глубокий тёмный интерфейс, вдохновленный северным сиянием. Доминируют оттенки глубокого синего, фиолетового и изумрудно-зеленого (цвета авроры). Стиль напоминает ночное небо в Арктике, тайну, космос и высокие технологии. Подходит для премиальных крипто-приложений, трейдинговых платформ или финтеха, который хочет создать ощущение «магии» и инноваций.

## Общая композиция

- **Sidebar:** 240px, тёмный
- **Header:** «Обзор»
- **Сетка:** 12 колонок
- **Hero:** Тёмный с aurora gradient effects

## Цветовая палитра

```
Фон основной:        #0F172A (deep dark blue)
Sidebar:             #1E293B
Карточки:            #1E293B
Hero gradient:       #0F172A с aurora overlay (зелёный + фиолетовый + синий)
Primary:             #8B5CF6 (violet)
Secondary:           #06B6D4 (cyan)
Accent:              #22C55E (aurora green)
Success:             #22C55E
Danger:              #EF4444
Текст основной:      #F1F5F9
Текст вторичный:     #94A3B8
Текст третичный:     #64748B
Border:              rgba(255, 255, 255, 0.08)
Shadow:              0 2px 12px rgba(0, 0, 0, 0.3)
```

## Hero-блок

- **Background:** #0F172A с multi-color aurora gradient overlay
- **Aurora effect:** Плавные gradient waves от зелёного к фиолетовому к синему
- **Баланс:** 36px/700, белый с glow
- **Radius:** 20px

## Карточки

- **Фон:** #1E293B
- **Border:** rgba(255,255,255,0.08)
- **Radius:** 16px
- **Hover:** border brightens

## Графики

- **Цвета:** Aurora palette — #8B5CF6, #06B6D4, #22C55E
- **Gradient lines:** Multi-color stroke

## Глубина

**Elevation + Aurora:**
- Aurora glow как ambient light
- Dark layers с soft shadows
- Cosmic, premium feel

## Свет

- Aurora glow — multiple colored light sources
- Subtle, ethereal
- Premium dark atmosphere

## Prompt для DeepSeek

> Создай dark dashboard в стиле Aurora Night. Фон #0F172A, карточки #1E293B. Hero с multi-color aurora gradient (зелёный → фиолетовый → синий). Баланс белый 36px с glow. Violet #8B5CF6, Cyan #06B6D4, Green #22C55E акценты. Тёмный, cosmic, premium. Radius 16px.

---

# Стиль №24 — Sand Dunes

## Общая концепция

Минималистичный, сухой и тёплый интерфейс, вдохновленный пустынными пейзажами и дюнами. Доминируют оттенки песочного, бежевого, светло-коричневого и терракотового. Стиль напоминает спокойствие пустыни, минимализм, архитектуру Ближнего Востока и натуральные материалы. Подходит для премиального сегмента, недвижимости или финтеха, который хочет подчеркнуть стабильность и «незыблемость».

## Общая композиция

- **Sidebar:** 240px, белый с sand undertone
- **Header:** «Обзор»
- **Hero:** Пустынная иллюстрация с дюнами

## Цветовая палитра

```
Фон основной:        #FEFCE8 (sand white)
Sidebar:             #FFFFFF
Hero gradient:       #FEF9C3 → #FEF08A (sandy)
Primary:             #CA8A04 (sand/gold)
Primary light:       rgba(202, 138, 4, 0.08)
Secondary:           #A16207
Success:             #22C55E
Danger:              #EF4444
Текст основной:      #1C1917
Текст вторичный:     #6B7280
Border:              rgba(202, 138, 4, 0.12)
Shadow:              0 2px 12px rgba(202, 138, 4, 0.08)
```

## Hero-блок

- **Background:** Sandy gradient #FEF9C3 → #FEF08A
- **Иллюстрация:** Песчаные дюны, возможно кactus или солнце
- **Баланс:** 32px/700
- **Radius:** 20px

## Карточки

- **Фон:** #FFFFFF
- **Border:** rgba(202,138,4,0.1)
- **Radius:** 14px

## Графики

- **Цвета:** #CA8A04, #EAB308, #FACC15 (gold/sand palette)

## Prompt для DeepSeek

> Создай пустынный dashboard в стиле Sand Dunes. Фон #FEFCE8, hero с gradient #FEF9C3 → #FEF08A и иллюстрацией песчаных дюн. Sand/gold акцент #CA8A04. Тёплый, zen-like, minimalist. Radius 14px.

---

# Стиль №25 — Midnight Sea

## Общая концепция

Глубокий, спокойный и таинственный интерфейс, вдохновленный ночным океаном. Доминируют оттенки очень тёмного синего, индиго и серебристо-белого (лунный свет). Стиль напоминает ночную вахту на корабле, спокойствие глубокого моря и звездное небо. Подходит для приложений, ориентированных на ночной режим, премиальный сегмент или финтеха, который хочет создать ощущение абсолютной надёжности и глубины.

## Общая композиция

- **Sidebar:** 240px, тёмный navy
- **Header:** «Обзор»
- **Hero:** Ночная морская иллюстрация с луной

## Цветовая палитра

```
Фон основной:        #0C1222 (deep navy)
Sidebar:             #111827
Карточки:            #1E293B
Hero gradient:       #0C1222 → #1E3A5F (midnight blue)
Primary:             #60A5FA (moonlight blue)
Primary light:       rgba(96, 165, 250, 0.15)
Secondary:           #94A3B8 (silver)
Accent:              #F8FAFC (moonlight white)
Success:             #22C55E
Danger:              #EF4444
Текст основной:      #F1F5F9
Текст вторичный:     #94A3B8
Текст третичный:     #64748B
Border:              rgba(255, 255, 255, 0.08)
Shadow:              0 2px 12px rgba(0, 0, 0, 0.4)
```

## Hero-блок

- **Background:** Midnight gradient #0C1222 → #1E3A5F
- **Иллюстрация:** Ночной океан, луна, отражение на воде, звёзды
- **Баланс:** 36px/700, белый с moonlight glow
- **Radius:** 20px

## Карточки

- **Фон:** #1E293B
- **Border:** rgba(255,255,255,0.08)
- **Radius:** 16px

## Графики

- **Цвета:** #60A5FA, #93C5FD, #BFDBFE (blue/moonlight palette)

## Глубина

**Elevation + Night:**
- Moonlight glow как ambient light
- Deep dark layers
- Calm, peaceful, mysterious

## Свет

- Moonlight — серебряный directional light
- Звёзды как tiny glow points
- Calm, nocturnal atmosphere

## Prompt для DeepSeek

> Создай ночной морской dashboard в стиле Midnight Sea. Фон #0C1222, карточки #1E293B. Hero с midnight gradient и иллюстрацией ночного океана с луной и звёздами. Moonlight blue акцент #60A5FA. Баланс белый с glow. Тёмный, calm, peaceful, mysterious. Radius 16px.

---

# Сравнительная таблица стилей

## Стили 1-15 (Базовые)

| № | Стиль | Фон | Primary | Glass | Radius | Sidebar | Для кого |
|---|-------|-----|---------|-------|--------|---------|----------|
| 1 | Premium Dark Neon | #0B1020 | #6E56CF | partial | 20px | 240px dark | Tech-savvy, молодёжь |
| 2 | Premium Light Organic | #F5F3EE | #2D7A3A | нет | 20px | 240px dark green | Эко-финтех, накопления |
| 3 | Dark Dashboard v2 | #0F1117 | #3B82F6 | нет | 12px | 220px dark | Аналитики, professional |
| 4 | Terminal Green | #0A0A0A | #00FF41 | нет | 0px | текстовый | Хакеры, крипто |
| 5 | Soft Glass Pastel | #F8F5FF | #A78BFA | full | 24px | glass | Широкая аудитория, женщины |
| 6 | Muted Light Premium | #FAFAF9 | #7C3AED | нет | 16px | 220px white | Премиум, private banking |
| 7 | Stripe Inspired | #0A0E27 | gradient | нет | 16px | 240px dark | Serious fintech |
| 8 | Linear Dark | #0D0D0D | #5E6AD2 | нет | 8px | 240px dark | Productivity, dev tools |
| 9 | Revolut Style | #191C1F | #0075EB | нет | 16px | 240px dark | Consumer banking |
| 10 | Wise Inspired | #FFFFFF | #9FE870 | нет | 12px | 240px white | International transfers |
| 11 | Glassmorphism Light | #E8ECF4 | #6366F1 | full | 24px | glass | Futuristic, elegant |
| 12 | Soft Neumorphism | #E4E9F0 | #6C63FF | нет | 20px | neumorphic | Taktil, modern |
| 13 | Gradient Hero | #F1F5F9 | #059669 | нет | 24px | white | Fresh, balance-focused |
| 14 | Minimal Monochrome | #FFFFFF | #000000 | нет | 12px | white | Ultra-minimalists |
| 15 | Compact Dashboard | #F8FAFC | #2563EB | нет | 8px | 200px | Power users, analysts |

## Стили 16-25 (Nature-themed)

| № | Стиль | Фон | Primary | Иллюстрация | Radius | Sidebar | Настроение |
|---|-------|-----|---------|-------------|--------|---------|------------|
| 16 | Sakura Bloom | #FFF5F7 | #F472B6 | Cherry blossom | 14px | 240px white | Spring, gentle, feminine |
| 17 | Sunrise Warmth | #FFFBF5 | #F59E0B | Sunrise/mountains | 14px | 240px white | Warm, optimistic, energetic |
| 18 | Ocean Breeze | #F0F9FF | #3B82F6 | Ocean/sailboat | 14px | 240px white | Fresh, spacious, calm |
| 19 | Lavender Dream | #FAF5FF | #8B5CF6 | Lavender fields | 14px | 240px white | Calm, harmonious, elegant |
| 20 | Mint Fresh | #F0FDF4 | #22C55E | Botanical leaves | 14px | 240px white | Fresh, clean, growth |
| 21 | Coral Reef | #FFF7ED | #F97316 | Underwater coral | 14px | 240px white | Playful, energetic, tropical |
| 22 | Autumn Leaves | #FFFBEB | #D97706 | Maple leaves | 14px | 240px white | Warm, cozy, comfort |
| 23 | Aurora Night | #0F172A | #8B5CF6 | Northern lights | 16px | 240px dark | Cosmic, premium, wonder |
| 24 | Sand Dunes | #FEFCE8 | #CA8A04 | Desert dunes | 14px | 240px white | Warm, zen, minimalist |
| 25 | Midnight Sea | #0C1222 | #60A5FA | Night ocean/moon | 16px | 240px dark | Calm, peaceful, mysterious |
