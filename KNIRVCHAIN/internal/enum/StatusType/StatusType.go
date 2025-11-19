package StatusType

type Int int

const (
	Unknown Int = iota
	Success
	Failed
	end
)

func IsValid(value int) bool {
	return value < int(end)
}
