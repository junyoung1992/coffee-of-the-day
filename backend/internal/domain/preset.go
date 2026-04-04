package domain

// Preset은 자주 사용하는 카페+메뉴 또는 원두+추출방식 조합의 공통 필드를 담는다.
type Preset struct {
	ID         string
	UserID     string
	Name       string
	LogType    LogType
	LastUsedAt *string
	CreatedAt  string
	UpdatedAt  string
}

// CafePresetDetail은 카페 프리셋의 전용 필드를 담는다.
type CafePresetDetail struct {
	CafeName    string
	CoffeeName  string
	TastingTags []string
}

// BrewPresetDetail은 홈브루 프리셋의 전용 필드를 담는다.
type BrewPresetDetail struct {
	BeanName     string
	BrewMethod   BrewMethod
	RecipeDetail *string
	BrewSteps    []string
}

// PresetFull은 공통 필드와 타입별 상세를 결합한다.
type PresetFull struct {
	Preset
	Cafe *CafePresetDetail
	Brew *BrewPresetDetail
}
