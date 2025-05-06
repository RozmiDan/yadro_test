package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/RozmiDan/yadro_test/internal/entity"
	"github.com/RozmiDan/yadro_test/internal/processor"
)

// rawConfig — промежуточный тип, точно повторяет JSON
type rawConfig struct {
	Laps        int     `json:"laps"`
	LapLen      float64 `json:"lapLen"`
	PenaltyLen  float64 `json:"penaltyLen"`
	FiringLines int     `json:"firingLines"`
	Start       string  `json:"start"`
	StartDelta  string  `json:"startDelta"`
}

func main() {
	// Читаем и конвертим конфиг
	cfg, err := readConfig(entity.CfgPath)
	if err != nil {
		log.Fatalf("readConfig: %v", err)
	}
	//fmt.Printf("config loaded: %+v\n", cfg)

	// Открываем файл с событиями
	f, err := os.Open(entity.EvPath)
	if err != nil {
		log.Fatalf("open events: %v", err)
	}
	defer f.Close()

	proc := processor.NewProcessor(cfg)

	// Сканируем построчно
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		ev, err := parseEvent(sc.Text())
		if err != nil {
			log.Fatalf("parseEvent [%s]: %v", sc.Text(), err)
		}
		resultString := proc.ProcessEvent(ev)
		fmt.Println(resultString)
	}
	if err := sc.Err(); err != nil {
		log.Fatalf("scan: %v", err)
	}

	report := proc.FinalReport()
	fmt.Println("Resulting table")
	for _, r := range report.Entries {

		lapStr := make([]string, cfg.Laps)
		for i := 0; i < cfg.Laps; i++ {
			if i < len(r.LapResults) {
				lr := r.LapResults[i]
				lapStr[i] = fmt.Sprintf("{%s, %.3f}", fmtDur(lr.Duration), lr.AvgSpeed)
			} else {
				lapStr[i] = "{,}"
			}
		}
		// штраф
		pen := r.PenaltyResult
		penStr := fmt.Sprintf("{%s, %.3f}", fmtDur(pen.TotalDuration), pen.AvgSpeed)

		// Hits/Shots
		shots := cfg.FiringLines * len(r.LapResults) * 5

		// итоговый вывод
		fmt.Printf("[%s] %d %v %s %d/%d\n",
			r.Status,
			r.CompetitorID,
			lapStr,
			penStr,
			r.Hits, shots,
		)
	}
}

// readConfig читает JSON, парсит строковые поля и конвертит их в типы
func readConfig(path string) (entity.Config, error) {
	var raw rawConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return entity.Config{}, err
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return entity.Config{}, err
	}

	// Парсим время старта
	// TODO
	var start time.Time
	if t, err := time.Parse(entity.TimeLayout, raw.Start); err == nil {
		start = t
	} else if t, err := time.Parse("15:04:05", raw.Start); err == nil {
		start = t
	} else {
		return entity.Config{}, fmt.Errorf("invalid start time %q", raw.Start)
	}

	parts := strings.Split(raw.StartDelta, ":")
	if len(parts) != 3 {
		return entity.Config{}, fmt.Errorf("invalid startDelta %q", raw.StartDelta)
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	s, _ := strconv.ParseFloat(parts[2], 64)
	delta := time.Duration(h)*time.Hour +
		time.Duration(m)*time.Minute +
		time.Duration(s*float64(time.Second))

	return entity.Config{
		Laps:        raw.Laps,
		LapLen:      raw.LapLen,
		PenaltyLen:  raw.PenaltyLen,
		FiringLines: raw.FiringLines,
		Start:       start,
		StartDelta:  delta,
	}, nil
}

// parseEvent парсит одну строку лога ивента
func parseEvent(line string) (entity.Event, error) {
	if len(line) < 2 || line[0] != '[' {
		return entity.Event{}, fmt.Errorf("bad format: %s", line)
	}

	idx := strings.IndexByte(line, ']')
	if idx == -1 {
		return entity.Event{}, fmt.Errorf("no closing ]: %s", line)
	}

	rawBracket := line[:idx+1]

	rest := strings.TrimSpace(line[idx+1:])

	parts := strings.SplitN(rest, " ", 3)
	if len(parts) < 2 {
		return entity.Event{}, fmt.Errorf("unexpected tokens: %q", rest)
	}

	id, err1 := strconv.Atoi(parts[0])
	ath, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return entity.Event{}, fmt.Errorf("bad ids in %q", rest)
	}

	extra := ""
	if len(parts) == 3 {
		extra = strings.TrimSpace(parts[2])
	}

	// парсим время внутри скобок
	t, err := time.Parse(entity.TimeLayout, line[1:idx])
	if err != nil {
		return entity.Event{}, fmt.Errorf("time parse: %w", err)
	}

	return entity.Event{
		Time:      t,
		ID:        id,
		AthleteID: ath,
		Extra:     extra,
		Raw:       rawBracket,
	}, nil
}

func fmtDur(d time.Duration) string {
	totalMs := d.Milliseconds()
	ms := totalMs % 1000
	totalSec := int(totalMs / 1000)
	sec := totalSec % 60
	totalMin := totalSec / 60
	min := totalMin % 60
	hr := totalMin / 60
	return fmt.Sprintf("%02d:%02d:%02d.%03d", hr, min, sec, ms)
}
