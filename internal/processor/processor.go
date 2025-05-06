package processor

import (
	"fmt"
	"sort"
	"time"

	"github.com/RozmiDan/yadro_test/internal/entity"
)

type Processor struct {
	cfg         entity.Config
	competitors map[int]*entity.Competitor
	outgoing    []entity.Event
	handlers    map[int]func(*entity.Competitor, entity.Event) string
}

func NewProcessor(cfg entity.Config) *Processor {
	p := &Processor{
		cfg:         cfg,
		competitors: make(map[int]*entity.Competitor),
		outgoing:    make([]entity.Event, 0),
		handlers:    make(map[int]func(*entity.Competitor, entity.Event) string),
	}

	// Регистрируем обработчики по ID
	p.handlers[1] = p.onRegistered
	p.handlers[2] = p.onStartTime
	p.handlers[3] = p.onAtStartLine
	p.handlers[4] = p.onStarted
	p.handlers[5] = p.onFiringRange
	p.handlers[6] = p.onHitTarget
	p.handlers[7] = p.onLeftFiringRange
	p.handlers[8] = p.onEnterPenalty
	p.handlers[9] = p.onLeavePenalty
	p.handlers[10] = p.onLapEnd
	p.handlers[11] = p.onCantContinue

	return p
}

func (p *Processor) ProcessEvent(ev entity.Event) string {
	c, ok := p.competitors[ev.AthleteID]
	if !ok {
		c = &entity.Competitor{
			ID:     ev.AthleteID,
			Status: "NotStarted",
		}
		p.competitors[ev.AthleteID] = c
	}

	var logLine string
	if h, found := p.handlers[ev.ID]; found {
		logLine = h(c, ev)
	} else {
		fmt.Printf("%s: unknown event %d for athlete %d\n", ev.Raw, ev.ID, ev.AthleteID)
	}
	return logLine
}

func (p *Processor) onRegistered(c *entity.Competitor, ev entity.Event) string {
	c.Registered = true
	return fmt.Sprintf("%s The competitor(%d) registered", ev.Raw, c.ID)
}

func (p *Processor) onStartTime(c *entity.Competitor, ev entity.Event) string {
	t, _ := time.Parse(entity.TimeLayout, ev.Extra)
	c.PlannedStart = t
	return fmt.Sprintf("%s The start time for the competitor(%d) was set by a draw to: %s", ev.Raw, c.ID, ev.Extra)
}

func (p *Processor) onAtStartLine(c *entity.Competitor, ev entity.Event) string {
	return fmt.Sprintf("%s The competitor(%d) is on the start line", ev.Raw, c.ID)
}

func (p *Processor) onStarted(c *entity.Competitor, ev entity.Event) string {
	c.ActualStart = ev.Time
	// проверка NotStarted
	if ev.Time.Sub(c.PlannedStart) > p.cfg.StartDelta {
		// c.Status = "NotStarted"
		out := fmt.Sprintf("%s The competitor(%d) NOT started in time", ev.Raw, c.ID)
		p.outgoing = append(p.outgoing, ev)
		return out
	}
	c.Status = "NotFinished"
	c.LapStart = ev.Time
	return fmt.Sprintf("%s The competitor(%d) has started", ev.Raw, c.ID)
}

func (p *Processor) onFiringRange(c *entity.Competitor, ev entity.Event) string {
	return fmt.Sprintf("%s The competitor(%d) is on firing range(%s)", ev.Raw, c.ID, ev.Extra)
}

func (p *Processor) onHitTarget(c *entity.Competitor, ev entity.Event) string {
	c.Hits++
	return fmt.Sprintf("%s The target (%s) has been hit by competitor(%d)", ev.Raw, ev.Extra, c.ID)
}

func (p *Processor) onLeftFiringRange(c *entity.Competitor, ev entity.Event) string {
	return fmt.Sprintf("%s The competitor(%d) left the firing range", ev.Raw, c.ID)
}

func (p *Processor) onEnterPenalty(c *entity.Competitor, ev entity.Event) string {
	c.PenaltyStart = ev.Time
	return fmt.Sprintf("%s The competitor(%d) entered the penalty laps", ev.Raw, c.ID)
}

func (p *Processor) onLeavePenalty(c *entity.Competitor, ev entity.Event) string {
	d := ev.Time.Sub(c.PenaltyStart)
	c.PenaltyTimes = append(c.PenaltyTimes, d)
	penSpeed := p.cfg.PenaltyLen / d.Seconds()
	c.PenaltySpeeds = append(c.PenaltySpeeds, penSpeed)
	return fmt.Sprintf("%s The competitor(%d) left the penalty laps", ev.Raw, c.ID)
}

func (p *Processor) onLapEnd(c *entity.Competitor, ev entity.Event) string {
	d := ev.Time.Sub(c.LapStart)
	c.LapTimes = append(c.LapTimes, d)
	c.LapStart = ev.Time
	speed := p.cfg.LapLen / d.Seconds()
	c.LapSpeeds = append(c.LapSpeeds, speed)

	var msg string

	if len(c.LapTimes) == p.cfg.Laps {
		c.Status = "Finished"
		p.outgoing = append(p.outgoing, ev)
		c.TotalTime = ev.Time.Sub(c.PlannedStart)
		msg = fmt.Sprintf("%s The competitor(%d) has finished", ev.Raw, c.ID)
	} else {
		msg = fmt.Sprintf("%s The competitor(%d) ended the main lap, duration=%s",
			ev.Raw, c.ID, d)
	}

	return msg
}

func (p *Processor) onCantContinue(c *entity.Competitor, ev entity.Event) string {
	c.Status = "NotFinished"
	p.outgoing = append(p.outgoing, ev)
	return fmt.Sprintf("%s The competitor(%d) can't continue: %s", ev.Raw, c.ID, ev.Extra)
}

func (p *Processor) FinalReport() entity.FinalReportList {
	entries := make([]entity.ReportEntry, 0, len(p.competitors))
	for _, c := range p.competitors {
		// Собираем LapResults
		laps := make([]entity.LapResult, len(c.LapTimes))
		for i := range c.LapTimes {
			laps[i] = entity.LapResult{
				Duration: c.LapTimes[i],
				AvgSpeed: p.cfg.LapLen / c.LapTimes[i].Seconds(),
			}
		}
		// Собираем PenaltyResult
		var sumPen time.Duration
		for _, pt := range c.PenaltyTimes {
			sumPen += pt
		}
		var avgPenSpeed float64
		if len(c.PenaltyTimes) > 0 {
			avgPenSpeed = p.cfg.PenaltyLen / sumPen.Seconds()
		}
		pen := entity.PenaltyResult{TotalDuration: sumPen, AvgSpeed: avgPenSpeed}

		entries = append(entries, entity.ReportEntry{
			CompetitorID:  c.ID,
			Status:        c.Status,
			TotalTime:     c.TotalTime,
			LapResults:    laps,
			PenaltyResult: pen,
			Hits:          c.Hits,
			Shots:         p.cfg.FiringLines * len(c.LapTimes) * 5,
		})
	}

	// 2) Сортируем
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		// сначала все не-финишировавшие
		aiDone, biDone := (a.Status == "Finished"), (b.Status == "Finished")
		if aiDone != biDone {
			return !aiDone
		}
		if aiDone {
			return a.TotalTime < b.TotalTime
		}
		return a.CompetitorID < b.CompetitorID
	})

	return entity.FinalReportList{
		Entries: entries,
	}
}
