package domain

// LogType distinguishes between cafe and home-brew coffee logs.
type LogType string

const (
	LogTypeCafe LogType = "cafe"
	LogTypeBrew LogType = "brew"
)

// RoastLevel indicates how darkly the beans were roasted.
type RoastLevel string

const (
	RoastLight  RoastLevel = "light"
	RoastMedium RoastLevel = "medium"
	RoastDark   RoastLevel = "dark"
)

// BrewMethod enumerates supported brewing methods.
type BrewMethod string

const (
	BrewMethodPourOver  BrewMethod = "pour_over"
	BrewMethodImmersion BrewMethod = "immersion"
	BrewMethodAeropress BrewMethod = "aeropress"
	BrewMethodEspresso  BrewMethod = "espresso"
	BrewMethodMokaPot   BrewMethod = "moka_pot"
	BrewMethodSiphon    BrewMethod = "siphon"
	BrewMethodColdBrew  BrewMethod = "cold_brew"
	BrewMethodOther     BrewMethod = "other"
)

// CoffeeLog holds the common fields shared by all coffee log entries.
type CoffeeLog struct {
	ID         string
	UserID     string
	RecordedAt string
	Companions []string
	LogType    LogType
	Memo       *string
	CreatedAt  string
	UpdatedAt  string
}

// CafeDetail holds fields specific to a cafe visit log.
type CafeDetail struct {
	CafeName    string
	Location    *string
	CoffeeName  string
	BeanOrigin  *string
	BeanProcess *string
	RoastLevel  *RoastLevel
	TastingTags []string
	TastingNote *string
	Impressions *string
	Rating      *float64
}

// BrewDetail holds fields specific to a home-brew log.
type BrewDetail struct {
	BeanName      string
	BeanOrigin    *string
	BeanProcess   *string
	RoastLevel    *RoastLevel
	RoastDate     *string
	TastingTags   []string
	TastingNote   *string
	BrewMethod    BrewMethod
	BrewDevice    *string
	CoffeeAmountG *float64
	WaterAmountMl *float64
	WaterTempC    *float64
	BrewTimeSec   *int
	GrindSize     *string
	BrewSteps     []string
	Impressions   *string
	Rating        *float64
}

// CoffeeLogFull combines the common log with its type-specific detail.
type CoffeeLogFull struct {
	CoffeeLog
	Cafe *CafeDetail
	Brew *BrewDetail
}
