package entity

import "time"

const (
	CfgPath    = "./config/config.json"
	EvPath     = "./config/events.log"
	TimeLayout = "15:04:05.000"
)

// Event — один входной ивент
type Event struct {
	Time      time.Time // время события
	ID        int       // eventID
	AthleteID int       // competitorID
	Extra     string    // доп значение
	Raw       string    // исходная строка
}

// Config — финальная структура с готовыми типами
type Config struct {
	Laps        int
	LapLen      float64
	PenaltyLen  float64
	FiringLines int
	Start       time.Time
	StartDelta  time.Duration
}

type Competitor struct {
	ID           int
	Registered   bool
	PlannedStart time.Time
	ActualStart  time.Time
	Status       string    // "" / "NotStarted" / "NotFinished" / "Finished"
	LapStart     time.Time // время старта текущего круга
	LapTimes     []time.Duration
	PenaltyStart time.Time // время входа в штрафной круг
	PenaltyTimes []time.Duration
	Hits         int
	Shots        int

	TotalTime time.Duration

	LapSpeeds     []float64 // для основного круга
	PenaltySpeeds []float64 // для штрафного круга
}

type ReportEntry struct {
    CompetitorID  int
    Status        string        // "Finished", "NotStarted", "NotFinished"
    TotalTime     time.Duration // полное время 
    LapResults    []LapResult   // детали по основным кругам
    PenaltyResult PenaltyResult // агрегированная инфа по штрафным кругам
    Hits          int           // всего попаданий
    Shots         int           // всего выстрелов
}

// Результат работы FinalReport
type FinalReportList struct {
    Entries     []ReportEntry // уже отсортированные
}

// Данные по одному кругу
type LapResult struct {
    Duration  time.Duration
    AvgSpeed  float64 
}

// Агрегированная статистика штрафных кругов
type PenaltyResult struct {
    TotalDuration time.Duration
    AvgSpeed      float64 
}
