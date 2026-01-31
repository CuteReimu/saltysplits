package main

import (
	"bufio"
	"encoding/xml"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"time"
)

var (
	run *xmlRun

	summary *SummaryData

	realTimeTotalData []TotalData
	gameTimeTotalData []TotalData
	realTimeReset     []ResetData
	gameTimeReset     []ResetData

	runBreakdownSegments []string
	runBreakdown         []*RunBreakdownData

	attempts = make(map[int]*xmlAttempt)
)

var (
	fileName = flag.String("f", "", "指定要分析的 .lss 文件路径")
)

func analysis() {
	var dummy string
	if *fileName != "" {
		dummy = *fileName
	} else {
		fmt.Println("将你的 .lss 文件拖进来，然后按回车键开始分析...")
		for {
			line, isPrefix, err := bufio.NewReader(os.Stdin).ReadLine()
			if err != nil {
				panic(err)
			}
			dummy += string(line)
			if !isPrefix {
				break
			}
		}
		dummy = strings.TrimSpace(dummy)
		if strings.HasPrefix(dummy, "\"") && strings.HasSuffix(dummy, "\"") {
			dummy = dummy[1 : len(dummy)-1]
		}
	}
	buf, err := os.ReadFile(dummy)
	if err != nil {
		panic(err)
	}
	err = xml.Unmarshal(buf, &run)
	if err != nil {
		panic(err)
	}

	analysisInfo()
	analysisTotal()
	analysisRun()

	fmt.Println("请打开浏览器访问 http://127.0.0.1:12334/ 查看分析结果")
	switch strings.ToLower(runtime.GOOS) {
	case "windows":
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", "http://127.0.0.1:12334/").Start()
	case "linux":
		_ = exec.Command("xdg-open", "http://127.0.0.1:12334/").Start()
	case "darwin":
		_ = exec.Command("open", "http://127.0.0.1:12334/").Start()
	}
}

func analysisInfo() {
	var playTime Duration
	bestTime := Duration(math.MaxInt64)
	for _, attempt := range run.Attempt {
		attempts[attempt.Id] = attempt
		if attempt.GameTime > 0 {
			bestTime = min(bestTime, attempt.GameTime)
		}
		playTime0 := max(attempt.RealTime, attempt.GameTime)
		if attempt.Started != "" && attempt.Ended != "" {
			started, err := time.Parse("01/02/2006 15:04:05", attempt.Started)
			if err != nil {
				panic(err)
			}
			ended, err := time.Parse("01/02/2006 15:04:05", attempt.Ended)
			if err != nil {
				panic(err)
			}
			playTime0 = max(playTime0, Duration(ended.Sub(started)))
		}
		playTime += playTime0
	}
	var sob Duration
	for _, seg := range run.Segments {
		var bestSegment = Duration(math.MaxInt64)
		if seg.BestSegmentTime.GameTime > 0 {
			bestSegment = seg.BestSegmentTime.GameTime
		} else if seg.BestSegmentTime.RealTime > 0 {
			bestSegment = seg.BestSegmentTime.RealTime
		}
		sob += bestSegment
	}
	summary = &SummaryData{
		BestTime:         bestTime,
		Sob:              sob,
		PossibleTimesave: bestTime - sob,
		Attempts:         run.AttemptCount,
		Playtime:         playTime,
	}
}

func analysisTotal() {
	seg := run.Segments[len(run.Segments)-1]
	for _, history := range seg.SegmentHistory {
		if attempt := attempts[history.Id]; attempt != nil {
			if attempt.RealTime > 0 {
				realTimeTotalData = append(realTimeTotalData, TotalData{attempt.Id, attempt.RealTime})
			}
			if attempt.GameTime > 0 {
				gameTimeTotalData = append(gameTimeTotalData, TotalData{attempt.Id, attempt.GameTime})
			}
		}
	}

	realResetCache := make(map[int]int) // attemptId -> 最后分段id
	gameResetCache := make(map[int]int) // attemptId -> 最后分段id
	for i, seg := range run.Segments {
		for _, history := range seg.SegmentHistory {
			if history.RealTime > 0 {
				realResetCache[history.Id] = i + 1
			}
			if history.GameTime > 0 {
				gameResetCache[history.Id] = i + 1
			}
		}
	}
	for i, seg := range run.Segments {
		var realCount, gameCount int
		for j := range run.Attempt {
			if realResetCache[j+1] == i {
				realCount++
			}
			if gameResetCache[j+1] == i {
				gameCount++
			}
		}
		if realCount > 0 {
			realTimeReset = append(realTimeReset, ResetData{seg.Name, realCount})
		}
		if gameCount > 0 {
			gameTimeReset = append(gameTimeReset, ResetData{seg.Name, gameCount})
		}
	}
}

func analysisRun() {
	type attemptTime struct {
		Time Duration
		Id   int
	}
	var m []attemptTime
	for _, attempt := range run.Attempt {
		if attempt.GameTime > 0 {
			m = append(m, attemptTime{attempt.GameTime, attempt.Id})
		}
	}
	slices.SortFunc(m, func(a, b attemptTime) int {
		return int(a.Time - b.Time)
	})
	if len(m) > 5 {
		m = m[:5]
	}
	for _, at := range m {
		data := &RunBreakdownData{
			Id: at.Id,
		}
		var acc Duration
		for _, seg := range run.Segments {
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
					Segment: seg.Name,
					Time:    int(time.Duration(acc).Seconds()),
				})
			}
		}
		runBreakdown = append(runBreakdown, data)
	}

	for _, seg := range run.Segments {
		runBreakdownSegments = append(runBreakdownSegments, seg.Name)
	}
}

func getSegment(index int) (*SegmentData, error) {
	if index < 0 || index >= len(run.Segments) {
		return nil, fmt.Errorf("index out of range")
	}
	seq := run.Segments[index]
	ret := &SegmentData{Min: Duration(math.MaxInt64)}
	var times []Duration
	var total Duration
	for _, history := range seq.SegmentHistory {
		t := history.GameTime
		if t == 0 {
			continue
		}
		times = append(times, t)
		total += t
		ret.Details = append(ret.Details, SegmentDetailData{
			Id:   history.Id,
			Time: int(time.Duration(t).Seconds()),
		})
		ret.Min = min(ret.Min, t)
		ret.Max = max(ret.Max, t)
	}
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

type SummaryData struct {
	BestTime         Duration
	Sob              Duration
	PossibleTimesave Duration
	Attempts         int
	Playtime         Duration
}

type TotalData struct {
	Id   int `json:"id"`
	Time Duration
}

type ResetData struct {
	Segment string
	Count   int
}

type RunBreakdownData struct {
	Id      int `json:"id"`
	Details []RunBreakdownDetailData
}

type RunBreakdownDetailData struct {
	Segment string `json:"y"`
	Time    int    `json:"x"`
}

type SegmentData struct {
	Min               Duration
	Max               Duration
	Average           Duration
	Median            Duration
	StandardDeviation int
	Details           []SegmentDetailData
}

type SegmentDetailData struct {
	Id   int `json:"id"`
	Time int
}
