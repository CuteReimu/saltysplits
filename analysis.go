package main

import (
	"errors"
	"math"
	"slices"
	"strings"
	"time"
)

const (
	// largeFileThreshold 统计的起始尝试 ID
	largeFileThreshold = 200
	// maxResetSegments 统计重试次数中显示的 TopN 分段数
	maxResetSegments = 14
	// topRunsCount 速通分析中显示的 TopN 最佳尝试数
	topRunsCount = 5
)

// Analyzer 保存分析的状态和结果.
type Analyzer struct {
	Run            *xmlRun
	StartAttemptID int

	Summary *SummaryData

	RealTimeTotalData     []TotalData
	GameTimeTotalData     []TotalData
	RealTimeReset         []ResetData
	GameTimeReset         []ResetData
	RealTimeResetBig      []ResetData
	GameTimeResetBig      []ResetData
	DisableShowBigSegment bool

	RunBreakdownSegments []string
	RunBreakdown         []*RunBreakdownData

	// attempts map keeps attempts for quick lookup
	attempts map[int]*xmlAttempt
}

// NewAnalyzer 创建一个新的 Analyzer 实例.
func NewAnalyzer(run *xmlRun, startAttemptId int) *Analyzer {
	return &Analyzer{
		Run:                   run,
		StartAttemptID:        startAttemptId,
		DisableShowBigSegment: true,
		attempts:              make(map[int]*xmlAttempt),
	}
}

// Analyze 执行所有分析任务.
func (a *Analyzer) Analyze() error {
	if err := a.analysisInfo(); err != nil {
		return err
	}

	a.analysisTotalData()
	a.analysisResetData()
	a.analysisRun()

	return nil
}

// GetSegment 计算特定分段的统计信息.
func (a *Analyzer) GetSegment(index int) (*SegmentData, error) {
	if index < 0 || index >= len(a.Run.Segments) {
		return nil, errors.New("index out of range")
	}

	seq := a.Run.Segments[index]
	ret := &SegmentData{Min: Duration(math.MaxInt64)}
	times := make([]Duration, 0, len(seq.SegmentHistory)-max(0, a.StartAttemptID))

	var total Duration
	for _, history := range seq.SegmentHistory {
		if history.Id < a.StartAttemptID {
			continue
		}

		t := history.GameTime
		if t == 0 {
			continue
		}

		times = append(times, t)
		total += t
		ret.Details = append(ret.Details, SegmentDetailData{
			Id:   history.Id,
			Time: time.Duration(t).Seconds(),
		})
		ret.Min = min(ret.Min, t)
		ret.Max = max(ret.Max, t)
	}

	slices.SortFunc(ret.Details, func(a, b SegmentDetailData) int {
		return a.Id - b.Id
	})

	ret.Average = Duration(math.Round(float64(total) / float64(len(times))))
	slices.Sort(times)

	if len(times)%2 == 1 {
		ret.Median = times[len(times)/2]
	} else {
		mid := len(times) / 2
		ret.Median = Duration(math.Round(float64(times[mid-1]+times[mid]) / 2))
	}

	var sdSum float64
	for _, t := range times {
		sdSum += math.Pow(float64(t)-float64(ret.Average), 2)
	}

	ret.StandardDeviation = int(time.Duration(math.Round(math.Sqrt(sdSum / float64(len(times))))).Seconds())

	return ret, nil
}

func (a *Analyzer) analysisInfo() error {
	var (
		playTime Duration
		bestTime Duration = math.MaxInt64
	)

	for _, attempt := range a.Run.Attempt {
		if attempt.Id < a.StartAttemptID {
			continue
		}

		a.attempts[attempt.Id] = attempt

		if attempt.GameTime > 0 {
			bestTime = min(bestTime, attempt.GameTime)
		}

		playTime0 := max(attempt.RealTime, attempt.GameTime)
		if attempt.Started != "" && attempt.Ended != "" {
			started, err := time.Parse("01/02/2006 15:04:05", attempt.Started)
			if err != nil {
				return err
			}

			ended, err := time.Parse("01/02/2006 15:04:05", attempt.Ended)
			if err != nil {
				return err
			}

			playTime0 = max(playTime0, Duration(ended.Sub(started)))
		}

		playTime += playTime0
	}

	var sob Duration

	for _, seg := range a.Run.Segments {
		var bestSegment = Duration(math.MaxInt64)
		if seg.BestSegmentTime.GameTime > 0 {
			bestSegment = seg.BestSegmentTime.GameTime
		} else if seg.BestSegmentTime.RealTime > 0 {
			bestSegment = seg.BestSegmentTime.RealTime
		}

		sob += bestSegment
	}

	a.Summary = &SummaryData{
		BestTime:         bestTime,
		Sob:              sob,
		PossibleTimesave: bestTime - sob,
		Attempts:         a.Run.AttemptCount - max(0, a.StartAttemptID),
		Playtime:         playTime,
	}

	return nil
}

func (a *Analyzer) analysisTotalData() {
	for _, attempt := range a.Run.Attempt {
		if attempt.Id < a.StartAttemptID {
			continue
		}

		if attempt.RealTime > 0 {
			a.RealTimeTotalData = append(a.RealTimeTotalData, TotalData{attempt.Id, time.Duration(attempt.RealTime).Seconds()})
		}

		if attempt.GameTime > 0 {
			a.GameTimeTotalData = append(a.GameTimeTotalData, TotalData{attempt.Id, time.Duration(attempt.GameTime).Seconds()})
		}
	}

	slices.SortFunc(a.RealTimeTotalData, func(a, b TotalData) int {
		return a.Id - b.Id
	})
	slices.SortFunc(a.GameTimeTotalData, func(a, b TotalData) int {
		return a.Id - b.Id
	})
}

func (a *Analyzer) analysisResetData() {
	var (
		realResetCache = make(map[int]int) // attemptId -> reset segment index
		gameResetCache = make(map[int]int) // attemptId -> reset segment index
	)
	for i, seg := range a.Run.Segments {
		for _, history := range seg.SegmentHistory {
			if history.Id < a.StartAttemptID {
				continue
			}

			// i + 1 because reset happens after this segment, meaning it reset on the NEXT segment.
			if history.RealTime > 0 {
				realResetCache[history.Id] = i + 1
			}

			if history.GameTime > 0 {
				gameResetCache[history.Id] = i + 1
			}
		}
	}

	var realCount, gameCount int
	for i, seg := range a.Run.Segments {
		realCount0, gameCount0 := a.getResetCount(realResetCache, gameResetCache, i)
		realCount, gameCount = realCount+realCount0, gameCount+gameCount0

		if realCount0 > 0 {
			a.RealTimeReset = append(a.RealTimeReset, ResetData{seg.Name, realCount0})
		}

		if gameCount0 > 0 {
			a.GameTimeReset = append(a.GameTimeReset, ResetData{seg.Name, gameCount0})
		}

		if strings.HasPrefix(seg.Name, "-") && i < len(a.Run.Segments)-1 {
			a.DisableShowBigSegment = false
			continue
		}

		if realCount > 0 {
			a.RealTimeResetBig = append(a.RealTimeResetBig, ResetData{seg.Name, realCount})
		}

		if gameCount > 0 {
			a.GameTimeResetBig = append(a.GameTimeResetBig, ResetData{seg.Name, gameCount})
		}

		realCount, gameCount = 0, 0
	}

	sortResetData(&a.RealTimeReset)
	sortResetData(&a.GameTimeReset)
	sortResetData(&a.RealTimeResetBig)
	sortResetData(&a.GameTimeResetBig)
}

func (a *Analyzer) getResetCount(realResetCache, gameResetCache map[int]int, segmentIndex int) (int, int) {
	var realCount, gameCount int
	for _, attempt := range a.Run.Attempt {
		if attempt.Id < a.StartAttemptID {
			continue
		}

		if realResetCache[attempt.Id] == segmentIndex {
			realCount++
		}

		if gameResetCache[attempt.Id] == segmentIndex {
			gameCount++
		}
	}

	return realCount, gameCount
}

func sortResetData(data *[]ResetData) {
	var otherCount int
	for len(*data) >= maxResetSegments {
		v := slices.MinFunc(*data, func(a, b ResetData) int {
			return a.Count - b.Count
		})
		minValue := v.Count

		*data = slices.DeleteFunc(*data, func(r ResetData) bool {
			if r.Count <= minValue {
				otherCount += r.Count
				return true
			}

			return false
		})
	}

	if otherCount > 0 {
		*data = append(*data, ResetData{
			Segment: "其它",
			Count:   otherCount,
		})
	}
}

func (a *Analyzer) analysisRun() {
	type attemptTime struct {
		Time Duration
		Id   int
	}

	var m []attemptTime

	for _, attempt := range a.Run.Attempt {
		if attempt.Id < a.StartAttemptID {
			continue
		}

		if attempt.GameTime > 0 {
			m = append(m, attemptTime{attempt.GameTime, attempt.Id})
		}
	}

	slices.SortFunc(m, func(a, b attemptTime) int {
		return int(a.Time - b.Time)
	})

	if len(m) > topRunsCount {
		m = m[:topRunsCount]
	}

	for _, at := range m {
		data := &RunBreakdownData{
			Id: at.Id,
		}

		var acc Duration
		for i, seg := range a.Run.Segments {
			var history *xmlAttempt
			for _, h := range seg.SegmentHistory {
				if h.Id == at.Id {
					history = h
					break
				}
			}

			if history != nil && history.GameTime > 0 {
				acc += history.GameTime
				data.Details = append(data.Details, RunBreakdownDetailData{
					Segment: i,
					Time:    time.Duration(acc).Seconds(),
				})
			}
		}

		a.RunBreakdown = append(a.RunBreakdown, data)
	}

	for _, seg := range a.Run.Segments {
		a.RunBreakdownSegments = append(a.RunBreakdownSegments, seg.Name)
	}
}

// SummaryData 保存运行的摘要统计信息.
type SummaryData struct {
	BestTime         Duration
	Sob              Duration
	PossibleTimesave Duration
	Attempts         int
	Playtime         Duration
}

// TotalData 代表特定尝试的总时间.
type TotalData struct {
	Id   int `json:"id"`
	Time float64
}

// ResetData 代表分段的重置计数.
type ResetData struct {
	Segment string
	Count   int
}

// RunBreakdownData 代表运行尝试的细分.
type RunBreakdownData struct {
	Id      int `json:"id"`
	Details []RunBreakdownDetailData
}

// RunBreakdownDetailData 代表运行细分中分段的时间细节.
type RunBreakdownDetailData struct {
	Segment int     `json:"y"`
	Time    float64 `json:"x"`
}

// SegmentData 代表单个分段的详细统计信息.
type SegmentData struct {
	Min               Duration
	Max               Duration
	Average           Duration
	Median            Duration
	StandardDeviation int
	Details           []SegmentDetailData
}

// SegmentDetailData 代表分段分析中特定尝试的时间.
type SegmentDetailData struct {
	Id   int `json:"id"`
	Time float64
}
