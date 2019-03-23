package module

type CalculateScore func(counts Counts) uint64 //计算组建分数的方法

// CalculateScoreSimple 代表简易的组件评分计算函数。
func CalculateScoreSimple(counts Counts) uint64 {
	return counts.CalledCount +
		counts.AcceptedCount<<1 +
		counts.CompletedCount<<2 +
		counts.HandlingNumber<<4
}

func SetScore(module *Module) bool {
	cal := (*module).ScoreCalculator()
	if cal == nil {
		cal = CalculateScoreSimple
	}
	newScore := cal((*module).Counts())

	if newScore == (*module).Score() {
		return false
	}
	(*module).SetScore(newScore)
	return true
}
