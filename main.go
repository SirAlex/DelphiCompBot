package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"log"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"time"
)

const rules = "Поддерживаемые команды:\n" +
	"/help - Показать список команд\n" +
	"/register - Пройти регистрацию\n" +
	"/confirm - Подтвердить регистрацию\n" +
	"/unregister - Отменить регистрацию\n" +
	"/status - Показать статус регистрации\n" +
	"/cancel - Отменить выполнение команды, например /register" +
	"/setbonus - Установить количество бонусов"

/*
help - Показать список команд
register - Пройти регистрацию
confirm - Подтверждение регистрации
unregister - Отменить регистрацию
status - Показать статус регистрации
cancel - Отменить выполнение команды
setbonus - Установить кол-во ваших бонусов
*/

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	log.Printf("DelphiCompBot starting...")

	db, err := gorm.Open("postgres", os.Getenv("DBCONNECTION"))
	if err != nil {
		panic("failed to connect database")
	}
	defer db.Close()

	log.Printf("Connected to DB, migrating...")

	// Migrate the schema
	db.AutoMigrate(&User{}, &RuBoardInfo{})

	log.Printf("Migration finished, connect to Telegram...")

	bot, err := tgbotapi.NewBotAPI(os.Getenv("TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = os.Getenv("DEBUG") != ""

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	var reply string
	var mkdown bool = false

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if reflect.TypeOf(update.Message.Text).Kind() == reflect.String && update.Message.Text != "" {
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			var user User
			if err := db.Preload("RBInfo").Find(&user, update.Message.From.ID).Error; err != nil {
				if gorm.IsRecordNotFoundError(err) {
					// Не нашли такого
					user.ID = update.Message.From.ID
					user.UserName = update.Message.Chat.UserName
					user.ChatID = update.Message.Chat.ID
					db.Create(&user)
				} else {
					reply = "Houston, we have a problem! Что то случилось с БД!"
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
					bot.Send(msg)
					continue
				}
			} else {
				// Update user info
				db.Model(&user).Updates(User{ChatID: update.Message.Chat.ID, UserName: update.Message.Chat.UserName})
			}

			if update.Message.IsCommand() {
				switch update.Message.Command() {
				case "start":
					// Send hello
					if user.Registered {
						reply = "Привет " + user.UserName + ", с возвращением!"
					} else {
						reply = "Привет! Для начала работы, Вам надо пройти регистрацию /register!\n\n" + rules
						mkdown = true
					}

				case "help":
					// Send help message
					if user.Registered {
						reply = rules
					} else {
						reply = "Для начала работы, вы должны пройти регистрацию /register.\n" +
							"После регистрации, вы сможете выполнить поиск по копилке, просто отправив часть названия компоненты.\n\n" +
							rules
						mkdown = true
					}

				case "status":
					// Send help message
					if user.Registered {
						reply = "Ваш статус: *Зарегистрирован*\n" +
							"Ru-board ник: *\"" + user.RBInfo.Login + "\"*\n" +
							"Регистрация пройдена: *" + user.RBInfo.RegisteredAt.Format(time.RFC822) + "*\n" +
							"Количество очков: *" + strconv.Itoa(user.RBInfo.TotalPoints) + "*, из них:\n" +
							" - За варез : *" + strconv.Itoa(user.RBInfo.WarezPoints) + "*\n" +
							" - За время : *" + strconv.Itoa(user.RBInfo.RegPoints) + "*\n" +
							" - Бонусов  : *" + strconv.Itoa(user.RBInfo.BonusPoints) + "*"
						mkdown = true
					} else {
						reply = "Ваш статус: *Регистрация не пройдена!*"
						mkdown = true
					}

				case "register":
					// Start registration procedure
					if user.Registered {
						reply = "Вы уже зарегистрированы, под ником: " + user.RBInfo.Login
					} else {
						db.Model(&user).Update("LastCommand", "register")
						reply = "Введите свой ник на форуме ru-board"
					}

				case "setbonus":
					// Start registration procedure
					if user.Registered {
						db.Model(&user).Update("LastCommand", "setbonus")
						reply = "Введите число ваших бонусов или отрицательное число, если у вас штрафов больше чем бонусов:"
					} else {
						reply = "Вы не зарегистрированы."
					}

				case "unregister":
					// Undo registration
					if user.Registered {
						db.Model(&user).Association("RBInfo").Clear()
						db.Model(&user).Update("Registered", false)
						reply = "OK, информация о вашей регистрации удалена"
					} else {
						reply = "Вы и так не зарегистрированы."
					}

				case "cancel":
					// Cancel pending command, such /register
					user.RBInfo.ConfirmationCode = ""
					user.RBInfo.ConfirmTryCount = 0
					user.LastCommand = ""
					db.Save(&user)
					reply = "OK, команда отменена"

				case "confirm":
					// Cancel pending command, such /register
					if !user.Registered {
						if user.RBInfo.ConfirmationCode != "" {
							db.Model(&user).Update("LastCommand", "confirm")
							reply = "Введите 5-буквенный код подтверждения, отправленный вам личным сообщением на форуме ru-board:"
						} else {
							reply = "Для начала, пройдите регистрацию /register"
						}
					} else {
						reply = "Вы уже зарегистрированы!"
					}

				case "test":
					// Cancel pending command, such /register
					if user.Registered {
						stats := GetUserPoints(user.RBInfo.Login)
						user.RBInfo.WarezPoints = stats.PointsWarez
						user.RBInfo.RegPoints = stats.PointsSinceRegDate
						user.RBInfo.RecalcPoints()
						db.Save(&user)
						reply = "У вас " + strconv.Itoa(user.RBInfo.TotalPoints) + " очков"
					} else {
						reply = "BAD, Вы не зарегистрированы"
					}
				case "testpm":
					// Cancel pending command, such /register
					user.RBInfo.ConfirmationCode = RandStringRunes(5)
					SendPrivateMessage("gruzd", "DelphiCompBot confirmation", "Your confirmation code is: "+user.RBInfo.ConfirmationCode)
					reply = "OK"

				default:
					reply = "Хм, такую команду я не знаю!"
				}
			} else {
				// It's not command
				switch user.LastCommand {
				case "confirm":
					if len(update.Message.Text) != 5 {
						reply = "Пожалуйста, введите код подтверждения, состоящий из 5 букв:"
					} else {
						if update.Message.Text == user.RBInfo.ConfirmationCode {
							now := time.Now()
							user.LastCommand = ""
							user.RBInfo.RegisteredAt = &now
							user.Registered = true
							user.RBInfo.ConfirmationCode = ""
							db.Save(&user)
							reply = "OK, Код подтверждения принят, теперь вы зарегистрированы в копилке. \nВы можете проверить свой стаус командой /status"
						} else {
							user.RBInfo.ConfirmTryCount += 1
							db.Save(&user)
							if user.RBInfo.ConfirmTryCount > 5 {
								user.LastCommand = ""
								user.RBInfo.ConfirmationCode = ""
								user.RBInfo.ConfirmTryCount = 0
								db.Save(&user)
								reply = "BAD, Вы превысили число возможных попыток ввести код подтверждения, вам надо заново пройти процесс регистрации."
							} else {
								reply = "BAD, Вы ввели неправильный код, попробуйте еще раз"
							}
						}
					}
				case "register":
					user.RBInfo.Login = update.Message.Text
					user.LastCommand = "confirm"
					user.RBInfo.ConfirmationCode = RandStringRunes(5)
					user.RBInfo.ConfirmTryCount = 0
					db.Save(&user)
					SendPrivateMessage(user.RBInfo.Login, "DelphiCompBot confirmation", "Your confirmation code is: "+user.RBInfo.ConfirmationCode)
					reply = "ОК, Будем считать, что ваш ник на форуме ru-board: '" + user.RBInfo.Login +
						"'. \nВам было отправлено личное сообщение на форуме. Вы должны завершить регистрацию, введя 5-буквенный код подтверждения:"

				case "setbonus":
					if bonus, err := strconv.Atoi(update.Message.Text); err != nil {
						reply = "BAD, ошибко чтения кол-ва бонусов, попробуйте еще раз указать *число* бонусов"
					} else {
						user.RBInfo.BonusPoints = bonus
						user.RBInfo.RecalcPoints()
						user.LastCommand = ""
						db.Save(&user)
						reply = "ОК, Кол-во ваших бонусов обновлено, теперь общее количество очков: *" + strconv.Itoa(user.RBInfo.TotalPoints) + "*"
						mkdown = true
					}
				default:
					// Считаем что любой текст - это запрос на поиск компоненты
					if user.Registered {
						reply = "Введите часть названия компоненты для поиска в копилке"
					} else {
						reply = "Что бы выполнить поиск, пройдите регистрацию! (/register)"
					}
				}
			}
		} else {
			reply = "Бот принимает только текстовые команды!"
		}
		log.Printf("[%s] Answer: %s", update.Message.From.UserName, reply)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
		if mkdown {
			msg.ParseMode = "markdown"
		}
		bot.Send(msg)
	}
}
