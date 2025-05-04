package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"
)

// Структура для парсинга XML ответа от API ЦБ РФ
type ValCurs struct {
	XMLName xml.Name `xml:"ValCurs"`
	Date    string   `xml:"Date,attr"`
	Valutes []Valute `xml:"Valute"`
}

type Valute struct {
	ID       string `xml:"ID,attr"`
	NumCode  string `xml:"NumCode"`
	CharCode string `xml:"CharCode"`
	Nominal  string `xml:"Nominal"`
	Name     string `xml:"Name"`
	Value    string `xml:"Value"`
}

// Структура для хранения информации о курсе валюты
type CurrencyRate struct {
	Date        time.Time
	CharCode    string
	Name        string
	Nominal     int
	Value       float64
	ValuePerOne float64 
}

// Структура для хранения минимального и максимального курса
type MinMaxRate struct {
	Currency CurrencyRate
	Value    float64
}

// Функция для получения данных за определенную дату
func fetchCurrencyData(date time.Time) ([]CurrencyRate, error) {
	dateStr := date.Format("02/01/2006")

	if date.After(time.Now()) {
		return nil, fmt.Errorf("невозможно получить данные для будущей даты: %s", date.Format("02.01.2006"))
	}

	url := fmt.Sprintf("http://www.cbr.ru/scripts/XML_daily_eng.asp?date_req=%s", dateStr)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка при создании запроса: %v", err)
	}

	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка при запросе к API: %v", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("некорректный статус ответа: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении ответа: %v", err)
	}

	// Проверяем, что ответ не пустой
	if len(body) == 0 {
		return nil, fmt.Errorf("получен пустой ответ")
	}

	// Создаем декодер с поддержкой windows-1251
	decoder := xml.NewDecoder(bytes.NewReader(body))
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		if strings.ToLower(charset) == "windows-1251" {
			return charmap.Windows1251.NewDecoder().Reader(input), nil
		}
		return input, nil
	}

	var valCurs ValCurs
	err = decoder.Decode(&valCurs)
	if err != nil {
		preview := string(body)
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		return nil, fmt.Errorf("ошибка при разборе XML: %v. Начало ответа: %s", err, preview)
	}

	rates := make([]CurrencyRate, 0, len(valCurs.Valutes))

	for _, valute := range valCurs.Valutes {
		value := strings.Replace(valute.Value, ",", ".", -1)
		nominal := strings.TrimSpace(valute.Nominal)

		valueFloat, err := strconv.ParseFloat(value, 64)
		if err != nil {
			fmt.Printf("Ошибка при парсинге значения '%s' для валюты %s: %v\n", value, valute.CharCode, err)
			continue
		}

		nominalInt, err := strconv.Atoi(nominal)
		if err != nil {
			fmt.Printf("Ошибка при парсинге номинала '%s' для валюты %s: %v\n", nominal, valute.CharCode, err)
			continue
		}

		// Рассчитываем курс за единицу валюты
		valuePerOne := valueFloat / float64(nominalInt)

		rate := CurrencyRate{
			Date:        date,
			CharCode:    valute.CharCode,
			Name:        valute.Name,
			Nominal:     nominalInt,
			Value:       valueFloat,
			ValuePerOne: valuePerOne,
		}

		rates = append(rates, rate)
	}

	return rates, nil
}

// Функция для получения данных за последние n дней
func fetchCurrencyDataForLastDays(days int) ([]CurrencyRate, error) {
	allRates := []CurrencyRate{}
	today := time.Now()

	maxAttempts := days * 2
	daysCollected := 0
	uniqueDates := make(map[string]bool)

	for i := 0; i < maxAttempts && daysCollected < days; i++ {
		date := today.AddDate(0, 0, -i)

		if date.After(today) {
			continue
		}

		dateKey := date.Format("2006-01-02")
		if uniqueDates[dateKey] {
			continue
		}

		rates, err := fetchCurrencyData(date)

		if err != nil {
			fmt.Printf("Предупреждение: не удалось получить данные за %s: %v\n", date.Format("02.01.2006"), err)
			continue
		}

		if len(rates) > 0 {
			allRates = append(allRates, rates...)
			uniqueDates[dateKey] = true
			daysCollected++

			fmt.Printf("Успешно получены данные за %s (%d из %d дней)\n", date.Format("02.01.2006"), daysCollected, days)
		} else {
			fmt.Printf("Предупреждение: нет данных о курсах валют за %s (возможно выходной или праздничный день)\n", date.Format("02.01.2006"))
		}

		if i < maxAttempts-1 && daysCollected < days {
			time.Sleep(300 * time.Millisecond)
		}
	}

	if len(allRates) == 0 {
		return nil, fmt.Errorf("не удалось получить данные о курсах валют за указанный период")
	}

	fmt.Printf("Всего получены данные за %d из %d запрошенных дней\n", daysCollected, days)
	return allRates, nil
}

// Функция поиска минимального и максимального курса среди всех валют
func findMinMaxRates(rates []CurrencyRate) (MinMaxRate, MinMaxRate) {
	minRate := MinMaxRate{Value: math.MaxFloat64}
	maxRate := MinMaxRate{Value: 0}

	for _, rate := range rates {
		if rate.ValuePerOne > maxRate.Value {
			maxRate.Value = rate.ValuePerOne
			maxRate.Currency = rate
		}

		if rate.ValuePerOne < minRate.Value {
			minRate.Value = rate.ValuePerOne
			minRate.Currency = rate
		}
	}

	return minRate, maxRate
}

// Функция расчёта среднего значения курса по всем валютам за период
func calculateAverageRate(rates []CurrencyRate) float64 {
	if len(rates) == 0 {
		return 0
	}

	currencyRates := make(map[string][]float64)

	for _, rate := range rates {
		currencyRates[rate.CharCode] = append(currencyRates[rate.CharCode], rate.ValuePerOne)
	}

	// Рассчитываем среднее значение для каждой валюты
	totalAvg := 0.0
	for _, values := range currencyRates {
		sum := 0.0
		for _, val := range values {
			sum += val
		}
		totalAvg += sum / float64(len(values))
	}

	// Общее среднее значение по всем валютам
	return totalAvg / float64(len(currencyRates))
}

// Функция для вывода информации о курсе
func printRateInfo(title string, rate MinMaxRate) {
	fmt.Printf("\n%s:\n", title)
	fmt.Printf("Валюта: %s (%s)\n", rate.Currency.Name, rate.Currency.CharCode)
	fmt.Printf("Курс: %.4f руб. за %d %s (%.4f руб. за единицу)\n",
		rate.Currency.Value, rate.Currency.Nominal, rate.Currency.CharCode, rate.Currency.ValuePerOne)
	fmt.Printf("Дата: %s\n", rate.Currency.Date.Format("02.01.2006"))
}

func main() {
	// Получение данных за последние 90 дней
	days := 90
	fmt.Printf("Получение данных о курсах валют за последние %d дней...\n", days)

	rates, err := fetchCurrencyDataForLastDays(days)
	if err != nil {
		fmt.Printf("Ошибка при получении данных: %v\n", err)
		return
	}

	if len(rates) == 0 {
		fmt.Println("Не удалось получить данные о курсах валют. Пожалуйста, проверьте подключение к интернету и доступность API ЦБ РФ.")
		return
	}

	// Вывод общей информации
	currencyCodes := make(map[string]bool)
	for _, rate := range rates {
		currencyCodes[rate.CharCode] = true
	}

	currencies := make([]string, 0, len(currencyCodes))
	for code := range currencyCodes {
		currencies = append(currencies, code)
	}
	sort.Strings(currencies)

	fmt.Printf("\nПолучены данные для %d валют: %s\n", len(currencies), strings.Join(currencies, ", "))
	fmt.Printf("Общее количество записей: %d\n", len(rates))

	// Нахождение минимального и максимального курса
	minRate, maxRate := findMinMaxRates(rates)

	// Проверка, что минимальный и максимальный курсы найдены
	if minRate.Value == math.MaxFloat64 || maxRate.Value == 0 {
		fmt.Println("Не удалось определить минимальный или максимальный курс валюты.")
		return
	}

	// Расчет среднего значения
	avgRate := calculateAverageRate(rates)

	// Вывод результатов
	printRateInfo("Максимальный курс", maxRate)
	printRateInfo("Минимальный курс", minRate)

	fmt.Printf("\nСреднее значение курса по всем валютам за %d дней: %.4f руб.\n", days, avgRate)

	fmt.Println("\nПрограмма успешно завершила работу.")
}
