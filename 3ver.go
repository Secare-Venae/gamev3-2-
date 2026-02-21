package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

//  КОНФИГУРАЦИЯ ИГРЫ  
const (
	START_HP          = 100
	START_MANA        = 50
	START_GOLD        = 100
	MANA_REGEN        = 10 // восстановление MP между боями
	HEAL_BETWEEN_BOSS = 30 // восстановление HP между боями
	SERVER_PORT       = "8080"
)

type BodyPart int

const (
	Head BodyPart = iota
	Torso
	Arms
	Legs
)

func (bp BodyPart) String() string {
	return []string{"голова", "торс", "руки", "ноги"}[bp]
}

type ItemType int

const (
	Weapon ItemType = iota
	Armor
	Consumable
	Special
)

type AbilityType int

const (
	DamageAbility AbilityType = iota
	HealAbility
	BuffAbility
)

// ==================== СЕТЕВЫЕ ТИПЫ ====================
type GameMessageType int

const (
	PlayerAction GameMessageType = iota
	PlayerReady
	GameStateMsg // Переименовано с GameState на GameStateMsg чтобы избежать конфликта
	ChatMessage
	Disconnect
)

type GameMessage struct {
	Type      GameMessageType
	PlayerID  int
	Action    string // "hit", "block", "ability", "item"
	HitPart   BodyPart
	BlockPart BodyPart
	AbilityID int
	ItemID    int
	Text      string      // для сообщений чата
	Player    *PlayerData // для передачи состояния игрока
}

type PlayerData struct {
	Name         string
	HP           int
	MaxHP        int
	Mana         int
	MaxMana      int
	BaseStrength int
	Gold         int
	Inventory    []Item
	Equipment    []Item
	Abilities    []Ability
}

// ==================== СТРУКТУРЫ ДАННЫХ ====================
type Ability struct {
	Name        string
	Description string
	Type        AbilityType
	Damage      int
	Heal        int
	ManaCost    int
	BuffAttack  int
	BuffDefense int
}

type Item struct {
	Name     string
	Type     ItemType
	Attack   int
	Defence  int
	PlusHP   int
	PlusMana int
	Price    int
}

type Character interface {
	GetName() string
	GetHP() int
	GetMana() int
	GetStrength() int
	SetHP(int)
	SetMana(int)
	Hit() BodyPart
	Block() BodyPart
	IsAlive() bool
	UseAbility(ability Ability, target Character) string
}

type Player struct {
	Name         string
	HP           int
	MaxHP        int
	Mana         int
	MaxMana      int
	Strength     int
	BaseStrength int
	Gold         int
	Inventory    []Item
	Equipment    []Item
	Abilities    []Ability
	ActiveBuffs  struct {
		AttackBuff  int
		DefenseBuff int
	}
}

type Enemy struct {
	Name       string
	HP         int
	Mana       int
	Strength   int
	Loot       []Item
	GoldDrop   int
	Ability    Ability
	DeathQuote string
}

type Merchant struct {
	Name     string
	Items    []Item
	Dialogue string
}

// ==================== РЕАЛИЗАЦИЯ МЕТОДОВ ====================
func (p *Player) GetName() string {
	return p.Name
}

func (p *Player) GetHP() int {
	return p.HP
}

func (p *Player) GetMana() int {
	return p.Mana
}

func (p *Player) GetStrength() int {
	totalStrength := p.BaseStrength + p.ActiveBuffs.AttackBuff
	// Добавляем бонусы от экипированного оружия
	for _, item := range p.Equipment {
		if item.Type == Weapon {
			totalStrength += item.Attack
		}
	}
	return totalStrength
}

func (p *Player) SetHP(hp int) {
	p.HP = hp
	if p.HP > p.MaxHP {
		p.HP = p.MaxHP
	}
}

func (p *Player) SetMana(mana int) {
	p.Mana = mana
	if p.Mana > p.MaxMana {
		p.Mana = p.MaxMana
	}
}

func (p *Player) Hit() BodyPart {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\nВыберите часть тела для удара:")
	fmt.Println("0 - голова")
	fmt.Println("1 - торс")
	fmt.Println("2 - руки")
	fmt.Println("3 - ноги")
	for {
		fmt.Print("Ваш выбор: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		choice, err := strconv.Atoi(input)
		if err == nil && choice >= 0 && choice <= 3 {
			return BodyPart(choice)
		}
		fmt.Println("Неверный выбор! Введите число от 0 до 3")
	}
}

func (p *Player) Block() BodyPart {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\nВыберите часть тела для защиты:")
	fmt.Println("0 - голова")
	fmt.Println("1 - торс")
	fmt.Println("2 - руки")
	fmt.Println("3 - ноги")
	for {
		fmt.Print("Ваш выбор: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		choice, err := strconv.Atoi(input)
		if err == nil && choice >= 0 && choice <= 3 {
			return BodyPart(choice)
		}
		fmt.Println("Неверный выбор! Введите число от 0 до 3")
	}
}

func (p *Player) IsAlive() bool {
	return p.HP > 0
}

func (p *Player) UseAbility(ability Ability, target Character) string {
	if p.Mana < ability.ManaCost {
		return "Недостаточно маны!"
	}
	p.Mana -= ability.ManaCost
	result := ""
	switch ability.Type {
	case DamageAbility:
		damage := ability.Damage + p.GetStrength()/2
		target.SetHP(target.GetHP() - damage)
		result = fmt.Sprintf("%s использует %s и наносит %d урона!", p.Name, ability.Name, damage)
	case HealAbility:
		heal := ability.Heal
		p.SetHP(p.HP + heal)
		result = fmt.Sprintf("%s использует %s и восстанавливает %d HP!", p.Name, ability.Name, heal)
	case BuffAbility:
		p.ActiveBuffs.AttackBuff += ability.BuffAttack
		p.ActiveBuffs.DefenseBuff += ability.BuffDefense
		result = fmt.Sprintf("%s использует %s! Атака +%d, Защита +%d",
			p.Name, ability.Name, ability.BuffAttack, ability.BuffDefense)
	}
	return result
}

func (e *Enemy) GetName() string {
	return e.Name
}

func (e *Enemy) GetHP() int {
	return e.HP
}

func (e *Enemy) GetMana() int {
	return e.Mana
}

func (e *Enemy) GetStrength() int {
	return e.Strength
}

func (e *Enemy) SetHP(hp int) {
	e.HP = hp
}

func (e *Enemy) SetMana(mana int) {
	e.Mana = mana
}

func (e *Enemy) Hit() BodyPart {
	return BodyPart(rand.Intn(4))
}

func (e *Enemy) Block() BodyPart {
	return BodyPart(rand.Intn(4))
}

func (e *Enemy) IsAlive() bool {
	return e.HP > 0
}

func (e *Enemy) UseAbility(ability Ability, target Character) string {
	if e.Mana < ability.ManaCost {
		return "У противника недостаточно маны!"
	}
	e.Mana -= ability.ManaCost
	result := ""

	switch ability.Type {
	case DamageAbility:
		damage := ability.Damage + e.Strength/2
		target.SetHP(target.GetHP() - damage)
		result = fmt.Sprintf("%s использует %s и наносит %d урона!", e.Name, ability.Name, damage)
	case HealAbility:
		heal := ability.Heal
		e.SetHP(e.HP + heal)
		result = fmt.Sprintf("%s использует %s и восстанавливает %d HP!", e.Name, ability.Name, heal)
	case BuffAbility:
		result = fmt.Sprintf("%s использует %s!", e.Name, ability.Name)
	}
	return result
}

// ==================== ИНВЕНТАРЬ И ЭКИПИРОВКА ====================
func (p *Player) TakeOff(i int) {
	if i < 0 || i >= len(p.Equipment) {
		fmt.Println("Неверный индекс предмета!")
		return
	}
	item := p.Equipment[i]
	p.Equipment = append(p.Equipment[:i], p.Equipment[i+1:]...)
	p.Inventory = append(p.Inventory, item)
	fmt.Printf("Вы сняли: %s\n", item.Name)
}

func (p *Player) Equip(i int) {
	if i < 0 || i >= len(p.Inventory) {
		fmt.Println("Неверный индекс предмета!")
		return
	}
	item := p.Inventory[i]
	if item.Type == Consumable {
		// расходный предмет
		p.SetHP(p.HP + item.PlusHP)
		p.SetMana(p.Mana + item.PlusMana)
		p.Inventory = append(p.Inventory[:i], p.Inventory[i+1:]...)
		fmt.Printf("Вы использовали %s!", item.Name)
		if item.PlusHP > 0 {
			fmt.Printf(" Восстановлено %d HP!", item.PlusHP)
		}
		if item.PlusMana > 0 {
			fmt.Printf(" Восстановлено %d маны!", item.PlusMana)
		}
		fmt.Println()
		return
	}
	// Проверяем тип предмета и есть ли уже такой тип в экипировке
	for _, equipped := range p.Equipment {
		if equipped.Type == item.Type {
			fmt.Printf("У вас уже экипирован предмет типа %s! Сначала снимите его.\n", getItemTypeName(item.Type))
			return
		}
	}
	// Перемещаем предмет из инвентаря в экипировку
	p.Inventory = append(p.Inventory[:i], p.Inventory[i+1:]...)
	p.Equipment = append(p.Equipment, item)
	fmt.Printf("Вы экипировали: %s\n", item.Name)
}

func (p *Player) ShowInventory() {
	fmt.Println("\n=== ИНВЕНТАРЬ ===")
	fmt.Printf("Золото: %d\n", p.Gold)
	if len(p.Inventory) == 0 {
		fmt.Println("Инвентарь пуст")
		return
	}
	for i, item := range p.Inventory {
		fmt.Printf("%d. %s", i, item.Name)
		switch item.Type {
		case Weapon:
			fmt.Printf(" (Оружие, +%d к атаке)", item.Attack)
		case Armor:
			fmt.Printf(" (Броня, +%d к защите)", item.Defence)
		case Consumable:
			fmt.Printf(" (Расходник")
			if item.PlusHP > 0 {
				fmt.Printf(", +%d HP", item.PlusHP)
			}
			if item.PlusMana > 0 {
				fmt.Printf(", +%d маны", item.PlusMana)
			}
			fmt.Printf(")")
		}
		fmt.Println()
	}
}

func (p *Player) ShowEquipment() {
	fmt.Println("\n=== ЭКИПИРОВКА ===")
	if len(p.Equipment) == 0 {
		fmt.Println("Нет экипированных предметов")
		return
	}
	for i, item := range p.Equipment {
		fmt.Printf("%d. %s", i, item.Name)
		switch item.Type {
		case Weapon:
			fmt.Printf(" (Оружие, +%d к атаке)", item.Attack)
		case Armor:
			fmt.Printf(" (Броня, +%d к защите)", item.Defence)
		}
		fmt.Println()
	}
}

func (p *Player) ShowAbilities() {
	fmt.Println("\n=== СПОСОБНОСТИ ===")
	for i, ability := range p.Abilities {
		fmt.Printf("%d. %s - %s (Стоимость маны: %d)\n",
			i, ability.Name, ability.Description, ability.ManaCost)
	}
}

// ==================== СИСТЕМА ТОРГОВЛИ ====================

func (m *Merchant) ShowItems(player *Player) {
	fmt.Printf("\n=== ЛАВКА %s ===\n", m.Name)
	fmt.Println(m.Dialogue)
	fmt.Printf("Ваше золото: %d\n", player.Gold)

	for i, item := range m.Items {
		fmt.Printf("%d. %s", i, item.Name)
		switch item.Type {
		case Weapon:
			fmt.Printf(" (Оружие, +%d к атаке)", item.Attack)
		case Armor:
			fmt.Printf(" (Броня, +%d к защите)", item.Defence)
		case Consumable:
			fmt.Printf(" (Расходник")
			if item.PlusHP > 0 {
				fmt.Printf(", +%d HP", item.PlusHP)
			}
			if item.PlusMana > 0 {
				fmt.Printf(", +%d маны", item.PlusMana)
			}
			fmt.Printf(")")
		}
		fmt.Printf(" - %d золота\n", item.Price)
	}
}

func (m *Merchant) BuyItem(player *Player, itemIndex int) {
	if itemIndex < 0 || itemIndex >= len(m.Items) {
		fmt.Println("Неверный индекс предмета!")
		return
	}

	item := m.Items[itemIndex]
	if player.Gold < item.Price {
		fmt.Println("Недостаточно золота!")
		return
	}

	player.Gold -= item.Price
	player.Inventory = append(player.Inventory, item)
	fmt.Printf("Вы купили %s за %d золота!\n", item.Name, item.Price)
}

// ==================== ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ====================

func getItemTypeName(itemType ItemType) string {
	return []string{"Оружие", "Броня", "Расходник", "Особый"}[itemType]
}

// Создание предметов
func createGameItems() []Item {
	return []Item{
		// Оружие
		{Name: "Змеиный клык", Type: Weapon, Attack: 18, Price: 125},
		{Name: "Ненасытный ятаган", Type: Weapon, Attack: 23, Price: 177},
		{Name: "Расколотое небо", Type: Weapon, Attack: 80, Price: 300},
		{Name: "Костолом", Type: Weapon, Attack: 40, Price: 200},
		{Name: "Танец смерти", Type: Weapon, Attack: 55, Price: 250},

		// Броня
		{Name: "Шипованный доспех", Type: Armor, Defence: 10, Price: 125},
		{Name: "Сияние пустоты", Type: Armor, Defence: 15, Price: 150},
		{Name: "Броня метревеца", Type: Armor, Defence: 20, Price: 177},
		{Name: "Облачение духов", Type: Armor, Defence: 30, Price: 200},
		{Name: "Кровавая кольчуга господина", Type: Armor, Defence: 50, Price: 300},

		// Расходники
		{Name: "Малое зелье здоровья", Type: Consumable, PlusHP: 20, Price: 20},
		{Name: "Большое зелье здоровья", Type: Consumable, PlusHP: 50, Price: 45},
		{Name: "Эликсир жизни", Type: Consumable, PlusHP: 100, Price: 80},
		{Name: "Малое зелье маны", Type: Consumable, PlusMana: 15, Price: 15},
		{Name: "Большое зелье маны", Type: Consumable, PlusMana: 30, Price: 30},
	}
}

// Создание способностей
func createAbilities() []Ability {
	return []Ability{
		{
			Name:        "Последний вздох",
			Description: "Подбрасывает врага и наносит 3 быстрых удара",
			Type:        DamageAbility,
			Damage:      100,
			ManaCost:    80,
		},
		{
			Name:        "Стальная буря",
			Description: "Делает выпал вперёд и наносит урон",
			Type:        DamageAbility,
			Damage:      10,
			ManaCost:    5,
		},
		{
			Name:        "Вестник заката",
			Description: "Бросает теневой клинок, который наносит урон",
			Type:        DamageAbility,
			Damage:      25,
			ManaCost:    15,
		},
		{
			Name:        "Клеймо смерти",
			Description: "Помечает врага меткой, кооторая наносит урон",
			Type:        DamageAbility,
			Damage:      40,
			ManaCost:    20,
		},
		{
			Name:        "Знак бури",
			Description: "Увеличивает атаку",
			Type:        BuffAbility,
			BuffAttack:  10,
			ManaCost:    10,
		},
		{
			Name:        "Храбрость",
			Description: "Увеличивает защиту",
			Type:        BuffAbility,
			BuffDefense: 10,
			ManaCost:    10,
		},
		{
			Name:        "Золотая эгида",
			Description: "Увеличивает атаку и защиту",
			Type:        BuffAbility,
			BuffAttack:  15,
			BuffDefense: 15,
			ManaCost:    20,
		},
		{
			Name:        "Исцеление",
			Description: "Восстанавливает здоровье",
			Type:        HealAbility,
			Heal:        25,
			ManaCost:    15,
		},
		{
			Name:        "Божественное исцеление",
			Description: "Сильное восстановление здоровья",
			Type:        HealAbility,
			Heal:        40,
			ManaCost:    30,
		},
	}
}

// Стартовый инвентарь
func getStartingInventory() []Item {
	return []Item{
		{Name: "Меч паладина", Type: Weapon, Attack: 5},
		{Name: "Доспех паладина", Type: Armor, Defence: 5},
		{Name: "Малое зелье здоровья", Type: Consumable, PlusHP: 20},
		{Name: "Малое зелье маны", Type: Consumable, PlusMana: 15},
	}
}

// Генерация лута для врага
func generateLoot() []Item {
	allItems := createGameItems()
	lootCount := rand.Intn(3) + 2 // 2-4 предмета
	loot := make([]Item, lootCount)

	for i := 0; i < lootCount; i++ {
		loot[i] = allItems[rand.Intn(len(allItems))]
	}

	return loot
}

// ==================== ИГРОВОЙ ПРОЦЕСС ====================

func fight(player Character, enemy Character) bool {
	reader := bufio.NewReader(os.Stdin)
	round := 1

	for player.IsAlive() && enemy.IsAlive() {
		fmt.Printf("\n=== РАУНД %d ===\n", round)
		fmt.Printf("%s: %d HP, %d маны\n", player.GetName(), player.GetHP(), player.GetMana())
		fmt.Printf("%s: %d HP, %d маны\n", enemy.GetName(), enemy.GetHP(), enemy.GetMana())

		// Ход игрока
		fmt.Println("\n--- Ваш ход ---")
		fmt.Println("1 - Обычная атака")
		fmt.Println("2 - Использовать способность")
		fmt.Println("3 - Показать способности")

		var playerHit, playerBlock BodyPart
		var abilityUsed bool

		for {
			fmt.Print("Ваш выбор: ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			switch input {
			case "1":
				playerHit = player.Hit()
				playerBlock = player.Block()
				abilityUsed = false
				break
			case "2":
				if p, ok := player.(*Player); ok {
					p.ShowAbilities()
					if len(p.Abilities) > 0 {
						fmt.Print("Выберите способность: ")
						abilityInput, _ := reader.ReadString('\n')
						abilityInput = strings.TrimSpace(abilityInput)
						if idx, err := strconv.Atoi(abilityInput); err == nil && idx >= 0 && idx < len(p.Abilities) {
							result := player.UseAbility(p.Abilities[idx], enemy)
							fmt.Println(result)
						}
					}
				}
				playerBlock = player.Block()
				abilityUsed = true
				break
			case "3":
				if p, ok := player.(*Player); ok {
					p.ShowAbilities()
				}
				continue
			default:
				fmt.Println("Неверный выбор!")
				continue
			}
			break
		}

		// Ход противника
		if !abilityUsed {
			enemyHit := enemy.Hit()
			enemyBlock := enemy.Block()

			fmt.Printf("\n%s бьет в %s и защищает %s\n",
				player.GetName(), playerHit, playerBlock)
			fmt.Printf("%s бьет в %s и защищает %s\n",
				enemy.GetName(), enemyHit, enemyBlock)

			// Обработка ударов
			if playerHit != enemyBlock {
				damage := player.GetStrength()
				enemy.SetHP(enemy.GetHP() - damage)
				fmt.Printf("%s наносит %d урона по %s!\n",
					player.GetName(), damage, enemy.GetName())
			} else {
				fmt.Printf("%s блокирует удар в %s!\n",
					enemy.GetName(), enemyBlock)
			}

			if enemyHit != playerBlock {
				damage := enemy.GetStrength()
				player.SetHP(player.GetHP() - damage)
				fmt.Printf("%s наносит %d урона по %s!\n",
					enemy.GetName(), damage, player.GetName())
			} else {
				fmt.Printf("%s блокирует удар в %s!\n",
					player.GetName(), playerBlock)
			}
		}

		round++

		if player.IsAlive() && enemy.IsAlive() {
			fmt.Print("\nНажмите Enter для продолжения...")
			reader.ReadString('\n')
		}
	}

	if player.IsAlive() {
		// Проверяем, есть ли у врага предсмертная цитата
		if e, ok := enemy.(*Enemy); ok && e.DeathQuote != "" {
			fmt.Printf("\n%s (хрипя): «%s»\n", e.Name, e.DeathQuote)
		}
		fmt.Printf("\n%s побеждает!\n", player.GetName())
		return true
	} else {
		fmt.Printf("\n%s побеждает!\n", enemy.GetName())
		return false
	}
}

// ==================== PVP ФУНКЦИИ (Локальные) ====================

func pvpFight(players []*Player) {
	reader := bufio.NewReader(os.Stdin)
	round := 1

	fmt.Println("\n=== НАЧАЛО PVP БИТВЫ ===")
	fmt.Printf("%s VS %s\n", players[0].Name, players[1].Name)
	fmt.Println("Битва идет до полной победы одного из игроков!")
	fmt.Println("Правила: Каждый ход игроки по очереди выбирают, куда атаковать и что защищать.")
	fmt.Println("После выбора обоих игроков происходит расчет урона.")
	fmt.Print("Нажмите Enter чтобы начать...")
	reader.ReadString('\n')

	for players[0].IsAlive() && players[1].IsAlive() {
		fmt.Printf("\n========== РАУНД %d ==========\n", round)
		fmt.Printf("%s: %d HP, %d маны | %s: %d HP, %d маны\n",
			players[0].Name, players[0].HP, players[0].Mana,
			players[1].Name, players[1].HP, players[1].Mana)

		// Ход первого игрока
		fmt.Printf("\n--- Ход %s ---\n", players[0].Name)
		fmt.Println("1 - Обычная атака")
		fmt.Println("2 - Использовать способность")
		fmt.Println("3 - Показать способности")
		fmt.Println("4 - Показать инвентарь")
		fmt.Println("5 - Использовать предмет из инвентаря")
		fmt.Println("6 - Отправить сообщение в чат")

		var player0Hit, player0Block BodyPart
		var player0AbilityUsed bool

		for {
			fmt.Printf("%s, ваш выбор: ", players[0].Name)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			switch input {
			case "1":
				fmt.Printf("\n%s, выберите куда атаковать:\n", players[0].Name)
				player0Hit = players[0].Hit()
				fmt.Printf("\n%s, выберите что защищать:\n", players[0].Name)
				player0Block = players[0].Block()
				player0AbilityUsed = false
				break
			case "2":
				players[0].ShowAbilities()
				if len(players[0].Abilities) > 0 {
					fmt.Print("Выберите способность: ")
					abilityInput, _ := reader.ReadString('\n')
					abilityInput = strings.TrimSpace(abilityInput)
					if idx, err := strconv.Atoi(abilityInput); err == nil && idx >= 0 && idx < len(players[0].Abilities) {
						if players[0].Mana >= players[0].Abilities[idx].ManaCost {
							result := players[0].UseAbility(players[0].Abilities[idx], players[1])
							fmt.Println(result)
							player0AbilityUsed = true
						} else {
							fmt.Println("Недостаточно маны!")
							continue
						}
					}
				}
				if !player0AbilityUsed {
					fmt.Printf("\n%s, выберите что защищать (после использования способности):\n", players[0].Name)
					player0Block = players[0].Block()
				}
				break
			case "3":
				players[0].ShowAbilities()
				continue
			case "4":
				players[0].ShowInventory()
				continue
			case "5":
				players[0].ShowInventory()
				if len(players[0].Inventory) > 0 {
					fmt.Print("Введите номер предмета для использования: ")
					itemInput, _ := reader.ReadString('\n')
					itemInput = strings.TrimSpace(itemInput)
					if idx, err := strconv.Atoi(itemInput); err == nil {
						players[0].Equip(idx)
					}
				}
				fmt.Printf("\n%s, выберите что защищать (после использования предмета):\n", players[0].Name)
				player0Block = players[0].Block()
				player0AbilityUsed = false
				break
			case "6":
				fmt.Print("Введите сообщение: ")
				msg, _ := reader.ReadString('\n')
				msg = strings.TrimSpace(msg)
				fmt.Printf("%s: %s\n", players[0].Name, msg)
				continue
			default:
				fmt.Println("Неверный выбор!")
				continue
			}
			break
		}

		fmt.Print("\nНажмите Enter для передачи хода второму игроку...")
		reader.ReadString('\n')

		// Ход второго игрока
		fmt.Printf("\n--- Ход %s ---\n", players[1].Name)
		fmt.Println("1 - Обычная атака")
		fmt.Println("2 - Использовать способность")
		fmt.Println("3 - Показать способности")
		fmt.Println("4 - Показать инвентарь")
		fmt.Println("5 - Использовать предмет из инвентаря")
		fmt.Println("6 - Отправить сообщение в чат")

		var player1Hit, player1Block BodyPart
		var player1AbilityUsed bool

		for {
			fmt.Printf("%s, ваш выбор: ", players[1].Name)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			switch input {
			case "1":
				fmt.Printf("\n%s, выберите куда атаковать:\n", players[1].Name)
				player1Hit = players[1].Hit()
				fmt.Printf("\n%s, выберите что защищать:\n", players[1].Name)
				player1Block = players[1].Block()
				player1AbilityUsed = false
				break
			case "2":
				players[1].ShowAbilities()
				if len(players[1].Abilities) > 0 {
					fmt.Print("Выберите способность: ")
					abilityInput, _ := reader.ReadString('\n')
					abilityInput = strings.TrimSpace(abilityInput)
					if idx, err := strconv.Atoi(abilityInput); err == nil && idx >= 0 && idx < len(players[1].Abilities) {
						if players[1].Mana >= players[1].Abilities[idx].ManaCost {
							result := players[1].UseAbility(players[1].Abilities[idx], players[0])
							fmt.Println(result)
							player1AbilityUsed = true
						} else {
							fmt.Println("Недостаточно маны!")
							continue
						}
					}
				}
				if !player1AbilityUsed {
					fmt.Printf("\n%s, выберите что защищать (после использования способности):\n", players[1].Name)
					player1Block = players[1].Block()
				}
				break
			case "3":
				players[1].ShowAbilities()
				continue
			case "4":
				players[1].ShowInventory()
				continue
			case "5":
				players[1].ShowInventory()
				if len(players[1].Inventory) > 0 {
					fmt.Print("Введите номер предмета для использования: ")
					itemInput, _ := reader.ReadString('\n')
					itemInput = strings.TrimSpace(itemInput)
					if idx, err := strconv.Atoi(itemInput); err == nil {
						players[1].Equip(idx)
					}
				}
				fmt.Printf("\n%s, выберите что защищать (после использования предмета):\n", players[1].Name)
				player1Block = players[1].Block()
				player1AbilityUsed = false
				break
			case "6":
				fmt.Print("Введите сообщение: ")
				msg, _ := reader.ReadString('\n')
				msg = strings.TrimSpace(msg)
				fmt.Printf("%s: %s\n", players[1].Name, msg)
				continue
			default:
				fmt.Println("Неверный выбор!")
				continue
			}
			break
		}

		// Обработка хода (расчет урона)
		fmt.Println("\n========== РЕЗУЛЬТАТЫ ХОДА ==========")

		// Атака первого игрока по второму
		if !player0AbilityUsed {
			fmt.Printf("\n%s атакует %s в %s\n", players[0].Name, players[1].Name, player0Hit)
			fmt.Printf("%s защищает %s\n", players[1].Name, player1Block)

			if player0Hit != player1Block {
				damage := players[0].GetStrength()
				players[1].SetHP(players[1].GetHP() - damage)
				fmt.Printf("💥 Удар достиг цели! %s наносит %d урона %s!\n",
					players[0].Name, damage, players[1].Name)
			} else {
				fmt.Printf("🛡️ %s блокирует удар в %s!\n",
					players[1].Name, player1Block)
			}
		}

		// Атака второго игрока по первому
		if !player1AbilityUsed {
			fmt.Printf("\n%s атакует %s в %s\n", players[1].Name, players[0].Name, player1Hit)
			fmt.Printf("%s защищает %s\n", players[0].Name, player0Block)

			if player1Hit != player0Block {
				damage := players[1].GetStrength()
				players[0].SetHP(players[0].GetHP() - damage)
				fmt.Printf("💥 Удар достиг цели! %s наносит %d урона %s!\n",
					players[1].Name, damage, players[0].Name)
			} else {
				fmt.Printf("🛡️ %s блокирует удар в %s!\n",
					players[0].Name, player0Block)
			}
		}

		// Итоги после хода
		fmt.Printf("\n--- ИТОГИ РАУНДА %d ---\n", round)
		fmt.Printf("%s: %d HP | %s: %d HP\n",
			players[0].Name, players[0].HP,
			players[1].Name, players[1].HP)

		round++

		if players[0].IsAlive() && players[1].IsAlive() {
			fmt.Print("\nНажмите Enter для следующего раунда...")
			reader.ReadString('\n')
		}
	}

	// Определение победителя
	fmt.Println("\n========== БИТВА ЗАВЕРШЕНА ==========")
	if players[0].IsAlive() {
		fmt.Printf("\n🏆 %s ПОБЕЖДАЕТ В PVP БИТВЕ! 🏆\n", players[0].Name)
		fmt.Printf("%s повержен! Поздравляем победителя!\n", players[1].Name)
	} else {
		fmt.Printf("\n🏆 %s ПОБЕЖДАЕТ В PVP БИТВЕ! 🏆\n", players[1].Name)
		fmt.Printf("%s повержен! Поздравляем победителя!\n", players[0].Name)
	}
}

func createPlayer(index int) *Player {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Введите имя %d-го игрока: ", index)
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	return &Player{
		Name:         name,
		HP:           START_HP,
		MaxHP:        START_HP,
		Mana:         START_MANA,
		MaxMana:      START_MANA,
		BaseStrength: 10,
		Strength:     10,
		Gold:         START_GOLD,
		Inventory:    getStartingInventory(),
		Equipment:    []Item{},
		Abilities:    []Ability{},
	}
}

// ==================== СЕТЕВЫЕ ФУНКЦИИ ====================

func playerToPlayerData(p *Player) *PlayerData {
	return &PlayerData{
		Name:         p.Name,
		HP:           p.HP,
		MaxHP:        p.MaxHP,
		Mana:         p.Mana,
		MaxMana:      p.MaxMana,
		BaseStrength: p.BaseStrength,
		Gold:         p.Gold,
		Inventory:    p.Inventory,
		Equipment:    p.Equipment,
		Abilities:    p.Abilities,
	}
}

func playerDataToPlayer(pd *PlayerData) *Player {
	return &Player{
		Name:         pd.Name,
		HP:           pd.HP,
		MaxHP:        pd.MaxHP,
		Mana:         pd.Mana,
		MaxMana:      pd.MaxMana,
		BaseStrength: pd.BaseStrength,
		Strength:     pd.BaseStrength,
		Gold:         pd.Gold,
		Inventory:    pd.Inventory,
		Equipment:    pd.Equipment,
		Abilities:    pd.Abilities,
	}
}

// Серверная часть
func runServer() {
	rand.Seed(time.Now().UnixNano())

	fmt.Println("=== ЗАПУСК СЕРВЕРА ===")
	fmt.Printf("Сервер запущен на порту %s. Ожидание подключения...\n", SERVER_PORT)

	ln, err := net.Listen("tcp", ":"+SERVER_PORT)
	if err != nil {
		fmt.Println("Ошибка запуска сервера:", err)
		return
	}
	defer ln.Close()

	conn, err := ln.Accept()
	if err != nil {
		fmt.Println("Ошибка принятия подключения:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Клиент подключился!")
	encoder := gob.NewEncoder(conn)
	decoder := gob.NewDecoder(conn)

	// Создание игрока 1 (сервер)
	fmt.Print("Введите ваше имя: ")
	reader := bufio.NewReader(os.Stdin)
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	player1 := &Player{
		Name:         name,
		HP:           START_HP,
		MaxHP:        START_HP,
		Mana:         START_MANA,
		MaxMana:      START_MANA,
		BaseStrength: 10,
		Strength:     10,
		Gold:         START_GOLD,
		Inventory:    getStartingInventory(),
		Equipment:    []Item{},
		Abilities:    []Ability{},
	}

	// Добавляем стартовые способности
	allAbilities := createAbilities()
	player1.Abilities = append(player1.Abilities, allAbilities[1], allAbilities[4], allAbilities[7])

	// Отправляем свои данные клиенту
	encoder.Encode(GameMessage{
		Type:   GameStateMsg,
		Player: playerToPlayerData(player1),
	})

	// Получаем данные игрока 2
	var msg GameMessage
	err = decoder.Decode(&msg)
	if err != nil {
		fmt.Println("Ошибка получения данных игрока 2:", err)
		return
	}

	player2 := playerDataToPlayer(msg.Player)
	fmt.Printf("\nИгрок 2 подключился: %s\n", player2.Name)

	fmt.Println("\n=== ИГРОКИ ГОТОВЫ ===")
	fmt.Printf("%s (Вы) VS %s\n", player1.Name, player2.Name)

	// Управление инвентарем перед боем
	fmt.Print("\nХотите управлять инвентарем перед боем? (y/n): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if strings.ToLower(input) == "y" {
		manageInventory(player1)
		// Отправляем обновленные данные
		encoder.Encode(GameMessage{
			Type:   GameStateMsg,
			Player: playerToPlayerData(player1),
		})
	}

	// Отправляем сигнал готовности
	encoder.Encode(GameMessage{Type: PlayerReady})

	// Ожидаем готовность клиента
	err = decoder.Decode(&msg)
	if err != nil || msg.Type != PlayerReady {
		fmt.Println("Ошибка ожидания готовности клиента")
		return
	}

	fmt.Println("Клиент готов! Начинаем бой...")
	fmt.Print("Нажмите Enter чтобы начать...")
	reader.ReadString('\n')

	// Запуск сетевого боя
	networkFight(player1, player2, encoder, decoder, true)
}

// Клиентская часть
func runClient() {
	rand.Seed(time.Now().UnixNano())

	fmt.Println("=== ПОДКЛЮЧЕНИЕ К СЕРВЕРУ ===")
	fmt.Print("Введите адрес сервера (например, localhost:8080): ")
	reader := bufio.NewReader(os.Stdin)
	address, _ := reader.ReadString('\n')
	address = strings.TrimSpace(address)

	if address == "" {
		address = "localhost:" + SERVER_PORT
	}

	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Println("Ошибка подключения к серверу:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Подключено к серверу!")
	encoder := gob.NewEncoder(conn)
	decoder := gob.NewDecoder(conn)

	// Получаем данные игрока 1
	var msg GameMessage
	err = decoder.Decode(&msg)
	if err != nil {
		fmt.Println("Ошибка получения данных игрока 1:", err)
		return
	}

	player1 := playerDataToPlayer(msg.Player)
	fmt.Printf("Противник: %s\n", player1.Name)

	// Создание игрока 2 (клиент)
	fmt.Print("Введите ваше имя: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	player2 := &Player{
		Name:         name,
		HP:           START_HP,
		MaxHP:        START_HP,
		Mana:         START_MANA,
		MaxMana:      START_MANA,
		BaseStrength: 10,
		Strength:     10,
		Gold:         START_GOLD,
		Inventory:    getStartingInventory(),
		Equipment:    []Item{},
		Abilities:    []Ability{},
	}

	// Добавляем стартовые способности
	allAbilities := createAbilities()
	player2.Abilities = append(player2.Abilities, allAbilities[1], allAbilities[4], allAbilities[7])

	// Отправляем свои данные серверу
	encoder.Encode(GameMessage{
		Type:   GameStateMsg,
		Player: playerToPlayerData(player2),
	})

	fmt.Println("\n=== ИГРОКИ ГОТОВЫ ===")
	fmt.Printf("%s (Вы) VS %s\n", player2.Name, player1.Name)

	// Управление инвентарем перед боем
	fmt.Print("\nХотите управлять инвентарем перед боем? (y/n): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if strings.ToLower(input) == "y" {
		manageInventory(player2)
		// Отправляем обновленные данные
		encoder.Encode(GameMessage{
			Type:   GameStateMsg,
			Player: playerToPlayerData(player2),
		})
	}

	// Ожидаем готовность сервера
	err = decoder.Decode(&msg)
	if err != nil || msg.Type != PlayerReady {
		fmt.Println("Ошибка ожидания готовности сервера")
		return
	}

	// Отправляем свою готовность
	encoder.Encode(GameMessage{Type: PlayerReady})

	fmt.Println("Сервер готов! Начинаем бой...")
	fmt.Print("Нажмите Enter чтобы начать...")
	reader.ReadString('\n')

	// Запуск сетевого боя
	networkFight(player2, player1, encoder, decoder, false)
}

// Сетевой бой
func networkFight(myPlayer, opponentPlayer *Player, encoder *gob.Encoder, decoder *gob.Decoder, isServer bool) {
	reader := bufio.NewReader(os.Stdin)
	round := 1
	myTurn := isServer // Сервер ходит первым

	// Канал для получения сообщений чата
	go func() {
		for {
			var msg GameMessage
			err := decoder.Decode(&msg)
			if err != nil {
				return
			}

			switch msg.Type {
			case ChatMessage:
				fmt.Printf("\n[ЧАТ] %s: %s\n", opponentPlayer.Name, msg.Text)
			case GameStateMsg:
				// Обновляем состояние противника
				if msg.Player != nil {
					opponentPlayer.HP = msg.Player.HP
					opponentPlayer.Mana = msg.Player.Mana
				}
			case Disconnect:
				fmt.Println("\nПротивник отключился!")
				return
			}
		}
	}()

	for myPlayer.IsAlive() && opponentPlayer.IsAlive() {
		fmt.Printf("\n========== РАУНД %d ==========\n", round)
		fmt.Printf("%s: %d HP, %d маны | %s: %d HP, %d маны\n",
			myPlayer.Name, myPlayer.HP, myPlayer.Mana,
			opponentPlayer.Name, opponentPlayer.HP, opponentPlayer.Mana)

		if myTurn {
			// Мой ход
			fmt.Printf("\n--- Ваш ход (%s) ---\n", myPlayer.Name)
			fmt.Println("1 - Обычная атака")
			fmt.Println("2 - Использовать способность")
			fmt.Println("3 - Показать способности")
			fmt.Println("4 - Показать инвентарь")
			fmt.Println("5 - Использовать предмет из инвентаря")
			fmt.Println("6 - Отправить сообщение в чат")

			var myHit, myBlock BodyPart
			var abilityUsed bool

			for {
				fmt.Print("Ваш выбор: ")
				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(input)

				switch input {
				case "1":
					fmt.Printf("\nВыберите куда атаковать:\n")
					myHit = myPlayer.Hit()
					fmt.Printf("\nВыберите что защищать:\n")
					myBlock = myPlayer.Block()
					abilityUsed = false

					// Отправляем действие на сервер
					encoder.Encode(GameMessage{
						Type:      PlayerAction,
						Action:    "hit",
						HitPart:   myHit,
						BlockPart: myBlock,
					})
					break
				case "2":
					myPlayer.ShowAbilities()
					if len(myPlayer.Abilities) > 0 {
						fmt.Print("Выберите способность: ")
						abilityInput, _ := reader.ReadString('\n')
						abilityInput = strings.TrimSpace(abilityInput)
						if idx, err := strconv.Atoi(abilityInput); err == nil && idx >= 0 && idx < len(myPlayer.Abilities) {
							if myPlayer.Mana >= myPlayer.Abilities[idx].ManaCost {
								result := myPlayer.UseAbility(myPlayer.Abilities[idx], opponentPlayer)
								fmt.Println(result)
								abilityUsed = true

								// Отправляем действие на сервер
								encoder.Encode(GameMessage{
									Type:      PlayerAction,
									Action:    "ability",
									AbilityID: idx,
									BlockPart: myBlock,
								})
							} else {
								fmt.Println("Недостаточно маны!")
								continue
							}
						}
					}
					if !abilityUsed {
						fmt.Printf("\nВыберите что защищать (после использования способности):\n")
						myBlock = myPlayer.Block()
					}
					break
				case "3":
					myPlayer.ShowAbilities()
					continue
				case "4":
					myPlayer.ShowInventory()
					continue
				case "5":
					myPlayer.ShowInventory()
					if len(myPlayer.Inventory) > 0 {
						fmt.Print("Введите номер предмета для использования: ")
						itemInput, _ := reader.ReadString('\n')
						itemInput = strings.TrimSpace(itemInput)
						if idx, err := strconv.Atoi(itemInput); err == nil {
							myPlayer.Equip(idx)

							// Отправляем действие на сервер
							encoder.Encode(GameMessage{
								Type:   PlayerAction,
								Action: "item",
								ItemID: idx,
							})
						}
					}
					fmt.Printf("\nВыберите что защищать (после использования предмета):\n")
					myBlock = myPlayer.Block()
					abilityUsed = false
					break
				case "6":
					fmt.Print("Введите сообщение: ")
					msg, _ := reader.ReadString('\n')
					msg = strings.TrimSpace(msg)

					// Отправляем сообщение
					encoder.Encode(GameMessage{
						Type: ChatMessage,
						Text: msg,
					})
					fmt.Printf("%s: %s\n", myPlayer.Name, msg)
					continue
				default:
					fmt.Println("Неверный выбор!")
					continue
				}
				break
			}

			// Отправляем обновленное состояние
			encoder.Encode(GameMessage{
				Type:   GameStateMsg,
				Player: playerToPlayerData(myPlayer),
			})

			fmt.Print("\nОжидание хода противника...")
		} else {
			// Ход противника
			fmt.Printf("\n--- Ход %s ---\n", opponentPlayer.Name)
			fmt.Println("Ожидание действий противника...")

			// Получаем действие противника
			var actionMsg GameMessage
			err := decoder.Decode(&actionMsg)
			if err != nil {
				fmt.Println("\nОшибка получения данных от противника")
				return
			}

			// Получаем обновленное состояние противника
			var stateMsg GameMessage
			err = decoder.Decode(&stateMsg)
			if err == nil && stateMsg.Type == GameStateMsg && stateMsg.Player != nil {
				opponentPlayer.HP = stateMsg.Player.HP
				opponentPlayer.Mana = stateMsg.Player.Mana
			}

			fmt.Println("Ход противника завершен!")
		}

		// Переключение хода
		myTurn = !myTurn

		if !myTurn {
			round++
		}

		if myPlayer.IsAlive() && opponentPlayer.IsAlive() {
			fmt.Print("\nНажмите Enter для продолжения...")
			reader.ReadString('\n')
		}
	}

	// Определение победителя
	fmt.Println("\n========== БИТВА ЗАВЕРШЕНА ==========")
	if myPlayer.IsAlive() {
		fmt.Printf("\n🏆 %s ПОБЕЖДАЕТ! 🏆\n", myPlayer.Name)
		fmt.Printf("%s повержен! Поздравляем победителя!\n", opponentPlayer.Name)
	} else {
		fmt.Printf("\n🏆 %s ПОБЕЖДАЕТ! 🏆\n", opponentPlayer.Name)
		fmt.Printf("%s повержен! Поздравляем победителя!\n", myPlayer.Name)
	}

	// Отправляем сигнал отключения
	encoder.Encode(GameMessage{Type: Disconnect})
}

// ==================== УПРАВЛЕНИЕ ИНВЕНТАРЕМ ====================

func manageInventory(player *Player) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\n=== УПРАВЛЕНИЕ ИНВЕНТАРЕМ ===")
		fmt.Println("1 - Показать инвентарь")
		fmt.Println("2 - Показать экипировку")
		fmt.Println("3 - Надеть предмет")
		fmt.Println("4 - Снять предмет")
		fmt.Println("5 - Показать способности")
		fmt.Println("6 - Вернуться к игре")

		fmt.Print("Ваш выбор: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			player.ShowInventory()
		case "2":
			player.ShowEquipment()
		case "3":
			player.ShowInventory()
			if len(player.Inventory) > 0 {
				fmt.Print("Введите номер предмета для экипировки: ")
				choice, _ := reader.ReadString('\n')
				choice = strings.TrimSpace(choice)
				if i, err := strconv.Atoi(choice); err == nil {
					player.Equip(i)
				}
			}
		case "4":
			player.ShowEquipment()
			if len(player.Equipment) > 0 {
				fmt.Print("Введите номер предмета для снятия: ")
				choice, _ := reader.ReadString('\n')
				choice = strings.TrimSpace(choice)
				if i, err := strconv.Atoi(choice); err == nil {
					player.TakeOff(i)
				}
			}
		case "5":
			player.ShowAbilities()
		case "6":
			return
		default:
			fmt.Println("Неверный выбор!")
		}
	}
}

func visitMerchant(player *Player, merchant Merchant) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\n=== ТОРГОВЛЯ ===")
		fmt.Println("1 - Показать товары")
		fmt.Println("2 - Купить предмет")
		fmt.Println("3 - Уйти")

		fmt.Print("Ваш выбор: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			merchant.ShowItems(player)
		case "2":
			merchant.ShowItems(player)
			if len(merchant.Items) > 0 {
				fmt.Print("Введите номер предмета для покупки: ")
				choice, _ := reader.ReadString('\n')
				choice = strings.TrimSpace(choice)
				if i, err := strconv.Atoi(choice); err == nil {
					merchant.BuyItem(player, i)
				}
			}
		case "3":
			return
		default:
			fmt.Println("Неверный выбор!")
		}
	}
}

// ==================== СЮЖЕТ И ГЛАВНЫЕ ФУНКЦИИ ====================

func showPrologue(playerName string) {
	fmt.Println("=== ПРОЛОГ ===")
	fmt.Printf("Мир Энтроса не просто умирает — он задыхается. Небеса затянуты «Ртутным Саваном», через который не пробивается ни солнечный свет, ни шепот богов.")
	fmt.Println("В этом мире плоть перестала быть стабильной.")
	fmt.Println("Из-за отсутствия божественного порядка всё живое подвергается Слиянию.")
	fmt.Println("Люди буквально врастают друг в друга, в камни или в собственные кошмары.")
	fmt.Println("Города превратились в огромные соборы из костей и застывшего крика.")
	fmt.Println("Жизнь здесь — это постоянная борьба за право остаться в своей собственной коже.")
	fmt.Println("Миром правит Конклав Гнилого Света — существа, которые обещают «Единство» (полное слияние всего живого в одну безмозглую массу), чтобы прекратить страдания.")
	fmt.Println("Корвус восстает против них не ради свободы, а из ненависти.")
	fmt.Println("Вы - \n", playerName)
	fmt.Println("Бывший инквизитор, чья единственная задача — охота на «Слитых».")
	fmt.Println("Чтобы иметь возможность убивать чудовищ, Вы используете «Орган-Паразит» — живое оружие")
	fmt.Println("Оно внедрено в ваше тело")
}

func showEpilogue(victory bool, playerName string) {
	fmt.Println("\n=== ЭПИЛОГ ===")
	if victory {
		fmt.Printf("%s, Вы достигаете Трона Савана и убивает Первородного Слитого. Вы ожидаете, что после этого «Ртутный Саван» рассеется и мир исцелится.\n", playerName)
		fmt.Println("Слияние было не проклятием, а единственным способом сохранить искры жизни в умирающей вселенной.")
		fmt.Println("Убив «Ядро», Вы не спасли мир — вы выдернули последнюю нить, державшую материю вместе.")
		fmt.Println("Вы стоите  на вершине горы трупов, смотря на то, как небо окончательно гаснет.")
		fmt.Println("Ваше тело окончательно мутирует, превращаясь в неподвижный памятник из плоти.")
		fmt.Println("Вы победили Конклав, сохранив свою индивидуальность,но платой за это стало полное одиночество в абсолютно мертвом мире, где больше нет ни боли, ни жизни.")
		fmt.Println("«Ты так боялся стать частью целого, что стал ничем».")
	} else {
		fmt.Printf("%s, Вы проиграли. Вы погибли", playerName)
		fmt.Println("Мы уверены, если вы попробуете снова, у вас получиться спасти мир!")
		fmt.Println("Вы ведь всё ещё хотите стать героем?")
	}
}

func showChapter(chapter int, description string) {
	fmt.Printf("\n=== ГЛАВА %d ===\n", chapter)
	fmt.Println(description)
	fmt.Println()
}

func main() {
	rand.Seed(time.Now().UnixNano())
	reader := bufio.NewReader(os.Stdin)

	// Регистрируем типы для gob
	gob.Register(&PlayerData{})
	gob.Register([]Item{})
	gob.Register([]Ability{})

	// Выбор режима игры
	fmt.Println("=== ВЫБОР РЕЖИМА ИГРЫ ===")
	fmt.Println("1 - Одиночная игра (PvE)")
	fmt.Println("2 - Мультиплеер")
	fmt.Print("Ваш выбор: ")
	modeInput, _ := reader.ReadString('\n')
	modeInput = strings.TrimSpace(modeInput)

	if modeInput == "2" {
		// Мультиплеер режим
		fmt.Println("\n=== МУЛЬТИПЛЕЕР ===")
		fmt.Println("1 - Горячий стул (на одном компьютере)")
		fmt.Println("2 - По сети")
		fmt.Print("Ваш выбор: ")
		multiInput, _ := reader.ReadString('\n')
		multiInput = strings.TrimSpace(multiInput)

		if multiInput == "2" {
			// Сетевой режим
			fmt.Println("\n=== СЕТЕВОЙ РЕЖИМ ===")
			fmt.Println("1 - Запустить сервер")
			fmt.Println("2 - Подключиться как клиент")
			fmt.Print("Ваш выбор: ")
			netInput, _ := reader.ReadString('\n')
			netInput = strings.TrimSpace(netInput)

			if netInput == "1" {
				runServer()
			} else {
				runClient()
			}
		} else {
			// Горячий стул
			fmt.Println("\n=== РЕЖИМ ГОРЯЧИЙ СТУЛ ===")
			fmt.Println("Создание двух игроков...")

			players := make([]*Player, 2)
			for i := 0; i < 2; i++ {
				players[i] = createPlayer(i + 1)
				// Добавляем стартовые способности
				allAbilities := createAbilities()
				players[i].Abilities = append(players[i].Abilities, allAbilities[1], allAbilities[4], allAbilities[7])
			}

			fmt.Println("\n=== ИГРОКИ СОЗДАНЫ ===")
			fmt.Printf("1. %s - HP: %d, Мана: %d, Сила: %d\n", players[0].Name, players[0].HP, players[0].Mana, players[0].GetStrength())
			fmt.Printf("2. %s - HP: %d, Мана: %d, Сила: %d\n", players[1].Name, players[1].HP, players[1].Mana, players[1].GetStrength())

			// Управление инвентарем перед боем для обоих игроков
			for i := 0; i < 2; i++ {
				fmt.Printf("\n--- Управление инвентарем для %s ---\n", players[i].Name)
				fmt.Print("Хотите управлять инвентарем перед боем? (y/n): ")
				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(input)

				if strings.ToLower(input) == "y" {
					manageInventory(players[i])
				}
			}

			// Начало PvP битвы
			pvpFight(players)
		}
	} else {
		// Одиночный режим (оригинальный сюжет)
		// Ввод имени игрока
		fmt.Print("Введите имя вашего персонажа: ")
		playerName, _ := reader.ReadString('\n')
		playerName = strings.TrimSpace(playerName)

		// Инициализация игрока
		player := &Player{
			Name:         playerName,
			HP:           START_HP,
			MaxHP:        START_HP,
			Mana:         START_MANA,
			MaxMana:      START_MANA,
			BaseStrength: 10,
			Strength:     10,
			Gold:         START_GOLD,
			Inventory:    getStartingInventory(),
			Equipment:    []Item{},
			Abilities:    []Ability{},
		}

		// Инициализация торговца
		merchant := Merchant{
			Name:     "Старый торговец",
			Dialogue: "Ты чего тут забыл?",
			Items: []Item{
				{Name: "Змеиный клык", Type: Weapon, Attack: 18, Price: 125},
				{Name: "Ненасытный ятаган", Type: Weapon, Attack: 23, Price: 177},
				{Name: "Божественная рапира", Type: Weapon, Attack: 80, Price: 300},
				{Name: "Меч павшего короля", Type: Weapon, Attack: 40, Price: 200},
				{Name: "Грань бесконечности", Type: Weapon, Attack: 55, Price: 250},

				{Name: "Шипованный доспех", Type: Armor, Defence: 10, Price: 125},
				{Name: "Сияние пустоты", Type: Armor, Defence: 15, Price: 150},
				{Name: "Броня метревеца", Type: Armor, Defence: 20, Price: 177},
				{Name: "Облачение духов", Type: Armor, Defence: 30, Price: 200},
				{Name: "Кровавая кольчуга господина", Type: Armor, Defence: 50, Price: 300},

				{Name: "Большое зелье здоровья", Type: Consumable, PlusHP: 50, Price: 45},
				{Name: "Большое зелье маны", Type: Consumable, PlusMana: 30, Price: 35},
			},
		}

		// Показываем пролог
		showPrologue(player.Name)

		// Главы игры
		chapters := []struct {
			StoryBefore string
			enemy       *Enemy
			newAbility  Ability
			StoryAfter  string
		}{
			{ // 1
				StoryBefore: "Вы достигаете Врат Опустевшего серебра. Небо здесь кажется тяжелее, чем обычно.\n" +
					"В тенях стоит фигура, которую вы узнали бы из тысячи. Это ваш учитель.\n" +
					"Но теперь его плоть колышется, словно живое море.\n" +
					"Бывший наставник Корвуса, который зашил себе глаза, чтобы не видеть ужаса Слияния, но его тело уже превратилось в массу переплетенных рук.\n" +
					"Сир Алдрих: «Я не вижу твоего лица, ученик... но я чую запах гнили, который ты называешь своей душой».",
				enemy: &Enemy{
					Name:       "Сир Алдрих Немигающий",
					HP:         50,
					Mana:       20,
					Strength:   8,
					Loot:       generateLoot(),
					GoldDrop:   25,
					DeathQuote: "Тьма, которую я выбрал... была милосерднее той, что несешь ты.",
				},
				newAbility: createAbilities()[0],
				StoryAfter: "Врата открыты. Впереди лишь запах гнили.",
			},
			{ //2
				StoryBefore: "Затопленные приюты Нижнего Города. Здесь Мать Гноя баюкает своих искаженных детей.\n" +
					"Женщина, которая «спасла» сирот, срастив их тела со своим, создав живой многоногий кокон.\n" +
					"«Тише, крошки... пришел злой человек, который хочет, чтобы мы снова были одиноки и холодны».",
				enemy: &Enemy{
					Name:       "Мать Гноя",
					HP:         80,
					Mana:       40,
					Strength:   12,
					Loot:       generateLoot(),
					GoldDrop:   50,
					DeathQuote: "Теперь... они наконец уснут.",
				},
				newAbility: createAbilities()[7],
				StoryAfter: "Тишина приюта пугает сильнее, чем крики.",
			},
			{ //3
				StoryBefore: "Пиршественный зал Эбеновой Крепости. Судья Варек ждет за накрытым столом из костей.\n" +
					"Существо, чей Орган-Паразит разросся до размеров комнаты, заставляя его постоянно пожирать плоть подданных\n" +
					"«Это не я бью тебя, малец... это Голод. И он говорит, что ты на вкус как старое, засохшее раскаяние».",
				enemy: &Enemy{
					Name:       "Судья Варек",
					HP:         110,
					Mana:       50,
					Strength:   18,
					Loot:       generateLoot(),
					GoldDrop:   70,
					DeathQuote: "«Наконец-то... тишина внутри... скоро ты услышишь их крики тоже».",
				},
				newAbility: createAbilities()[2],
				StoryAfter: "Вы переступаете через объедки его величия.",
			},
			{ // 4
				StoryBefore: "Мост Вздохов. Близнецы Раздора: Каин и Авель, Два рыцаря, сросшиеся спинами. Они ненавидят друг друга, но вынуждены сражаться как одно целое.\n" +
					"«Мы были братьями... но теперь я чувствую только ненависть к тебе, который так и не смог стать мной».\n" +
					"«Мы — легион из двоих. Мы ненавидим каждый вдох друг друга, но ты... ты даже не половина человека».",
				enemy: &Enemy{
					Name:       "Близнецы Раздора",
					HP:         140,
					Mana:       60,
					Strength:   22,
					Loot:       generateLoot(),
					GoldDrop:   100,
					DeathQuote: "Свободен... как же холодно быть одному...",
				},
				newAbility: createAbilities()[6],
				StoryAfter: "Они наконец едины в смерти.",
			},
			{ //5
				StoryBefore: "Сад Освежеванных Роз. Тень Искупления: Иеремия Безмолвный - Бывший святой, который вывернул свою кожу наизнанку, пытаясь «вывернуться» из проклятия Слияния.\n" +
					"«Зачем ты сопротивляешься плоти? Она — единственное, что у тебя осталось. Твои мысли уже давно принадлежат не тебе».",
				enemy: &Enemy{
					Name:       "Иеремия Безмолвный",
					HP:         170,
					Mana:       80,
					Strength:   28,
					Loot:       generateLoot(),
					GoldDrop:   130,
					DeathQuote: "Убей меня... вырежи мое имя из этой истории.",
				},
				newAbility: createAbilities()[8],
				StoryAfter: "Лепестки роз пропитались его кровью.",
			},
			{
				StoryBefore: "Обсерватория Шепотов. Великий Архитектор: Консул Малакай - Главный идеолог Конклава, превративший себя в живую библиотеку из плоти и пергамента.\n" +
					"«Ты пришел сломать идеальный механизм ради своего жалкого 'Я'? Какое высокомерие для куска гниющего мяса».",
				enemy: &Enemy{
					Name:       "Консул Малакай",
					HP:         210,
					Mana:       100,
					Strength:   35,
					Loot:       generateLoot(),
					GoldDrop:   200,
					DeathQuote: "Ты... всего лишь лишняя запятая в моем черновике.«Стань частью нас! Стань вечным! Стань никем!»",
				},
				newAbility: createAbilities()[3],
				StoryAfter: "Книги сгорели. Остались лишь вы.",
			},
			{
				StoryBefore: "Трон Немого Неба. Перед вами стоит ваше собственное Отражение.\n" +
					"Это не физический враг, а визуализация Органа-Паразита внутри Вас, принявшая облик самого героя в его лучшей версии.\n" +
					"«Посмотри на меня. Я — твоя память. Я — то, что ты убивал каждый раз, когда нажимал на 'курок'».",
				enemy: &Enemy{
					Name:       "Отражение",
					HP:         300,
					Mana:       150,
					Strength:   45,
					Loot:       generateLoot(),
					GoldDrop:   500,
					DeathQuote: "Ты победил. Ты один. Теперь тишина принадлежит тебе.",
				},
				newAbility: createAbilities()[0],
				StoryAfter: "«Ты победил. Ты один. Теперь тишина будет принадлежать только тебе. Навсегда».\n" +
					"Мир замер в ожидании финала.",
			},
		}

		victory := true

		// Игровой цикл по главам
		for chapter, data := range chapters {
			fmt.Printf("\n=== ГЛАВА %d ===\n", chapter+1)
			fmt.Println(data.StoryBefore)

			// Предлагаем посетить торговца
			fmt.Print("Хотите посетить торговца перед боем? (y/n): ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if strings.ToLower(input) == "y" {
				visitMerchant(player, merchant)
			}

			// Управление инвентарем перед боем
			fmt.Print("Хотите управлять инвентарем перед боем? (y/n): ")
			input, _ = reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if strings.ToLower(input) == "y" {
				manageInventory(player)
			}

			fmt.Printf("\nПриготовьтесь к бою с %s!\n", data.enemy.GetName())
			fmt.Print("Нажмите Enter чтобы начать бой...")
			reader.ReadString('\n')

			// Бой
			if !fight(player, data.enemy) {
				victory = false
				break
			}

			// Получение трофеев
			fmt.Printf("\n=== ТРОФЕИ ===\n")
			fmt.Printf("Вы получаете %d золота!\n", data.enemy.GoldDrop)
			player.Gold += data.enemy.GoldDrop

			for _, item := range data.enemy.Loot {
				fmt.Printf("Вы получаете: %s!\n", item.Name)
				player.Inventory = append(player.Inventory, item)
			}

			// Получение новой способности
			fmt.Printf("\n=== НОВАЯ СПОСОБНОСТЬ ===\n")
			fmt.Printf("Вы изучили: %s - %s\n", data.newAbility.Name, data.newAbility.Description)
			player.Abilities = append(player.Abilities, data.newAbility)

			// Восстановление между боями
			player.SetHP(player.GetHP() + HEAL_BETWEEN_BOSS)
			player.SetMana(player.GetMana() + MANA_REGEN)
			fmt.Printf("Вы восстановили %d HP и %d маны. Теперь у вас %d HP и %d маны\n",
				HEAL_BETWEEN_BOSS, MANA_REGEN, player.GetHP(), player.GetMana())

			if data.StoryAfter != "" {
				fmt.Println("\n" + data.StoryAfter)
			}

			if chapter < len(chapters)-1 {
				fmt.Print("\nНажмите Enter чтобы продолжить путешествие...")
				reader.ReadString('\n')
			}
		}

		// Показываем эпилог
		showEpilogue(victory, player.Name)

		if victory {
			fmt.Println("\n🎉 ПОЗДРАВЛЯЕМ! ВЫ ПРОШЛИ ИГРУ! 🎉")
		} else {
			fmt.Println("\n💀 ИГРА ОКОНЧЕНА. ПОПРОБУЙТЕ СНОВА! 💀")
		}
	}
}
