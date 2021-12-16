package structs

import "time"

// Structs according to Database (Snake_case)

type User struct {
	Id           int
	Name         string `json:"name"`
	Email        string `json:"email"`
	Telephone    string `json:"telephone"`
	Address      string `json:"address"`
	Company_name string `json:"company_name"`
	Password     string `json:"password"`
	Admin        int
	Created_date time.Time
	Updated_date time.Time
}

type Trash_version struct {
	Id                   int
	Version_name         string
	Inorganic_max_height int
	Organic_max_height   int
}

// Prevent GORM from adding "s" at the end of `trash_version` table name.
func (trash_version *Trash_version) TableName() string {
	return "trash_version"
}

type Trash struct {
	Id                     int       `json:"id"`
	Trash_code             string    `json:"trash_code"`
	Assigned               string    `json:"assigned"`
	Created_date           time.Time `json:"created_date"`
	Assigned_date          time.Time `json:"assigned_date"`
	User_id                int       `json:"user_id"`
	Latitude               string    `json:"latitude"`
	Longitude              string    `json:"longitude"`
	Location               string    `json:"location"`
	Guarantee_expired_date time.Time `json:"guarantee_expired_date"`
	Trash_version_id       int       `json:"trash_version_id"`
	Custom_name            string    `json:"custom_name"`
}

type Trash_capacity struct {
	Id                 int
	Trash_id           int
	Organic_capacity   int
	Inorganic_capacity int
	Created_at         time.Time
}

type Trash_reading struct {
	Id           int
	Trash_id     int
	Category     string
	Type         string
	Created_date time.Time
}

// Structs for responses (camelCase)

type TrashCapacity struct {
	Trash_id             int
	Organic_capacity     int
	Inorganic_capacity   int
	Organic_max_height   int
	Inorganic_max_height int
}

type TrashLogs struct {
	Trash_can_id  int
	Trash_reading []Trash_reading
}

type TrashReading struct {
	Trash_sorter_name     string
	Trash_sorter_location string
	Total                 int
}

type SummaryResponse struct {
	Type     map[string]int
	Category map[string]int
}

type TypeChart struct {
	ChartKey map[time.Time]map[string]int
}

type TypeChartResponse struct {
	Created_date time.Time
	Data_type    []map[string]string
}

type TrashAdd struct {
	Trash_code             string
	Trash_version_id       int
}