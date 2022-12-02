package dashboards

import "time"

// Dashboard represents information about a New Relic dashboard.
type Dashboard struct {
	ID              int                 `json:"id"`
	Title           string              `json:"title,omitempty"`
	Icon            DashboardIconType   `json:"icon,omitempty"`
	CreatedAt       time.Time           `json:"created_at,omitempty"`
	UpdatedAt       time.Time           `json:"updated_at,omitempty"`
	Visibility      VisibilityType      `json:"visibility,omitempty"`
	Editable        EditableType        `json:"editable,omitempty"`
	UIURL           string              `json:"ui_url,omitempty"`
	APIURL          string              `json:"api_url,omitempty"`
	OwnerEmail      string              `json:"owner_email,omitempty"`
	Metadata        DashboardMetadata   `json:"metadata"`
	Filter          DashboardFilter     `json:"filter,omitempty"`
	Widgets         []DashboardWidget   `json:"widgets,omitempty"`
	GridColumnCount GridColumnCountType `json:"grid_column_count,omitempty"`
}

// GridColumnCountType represents an option for the dashboard's grid column count.
// New Relic Insights supports a 3 column grid.
// New Relic One supports a 12 column grid.
type GridColumnCountType int

var (
	GridColumnCountTypes = struct {
		Insights GridColumnCountType
		One      GridColumnCountType
	}{
		Insights: 3,
		One:      12,
	}
)

// VisibilityType represents an option for the dashboard's visibility field.
type VisibilityType string

var (
	// VisibilityTypes specifies the possible options for a dashboard's visibility.
	VisibilityTypes = struct {
		Owner VisibilityType
		All   VisibilityType
	}{
		Owner: "owner",
		All:   "all",
	}
)

// EditableType represents an option for the dashboard's editable field.
type EditableType string

var (
	// EditableTypes specifies the possible options for who can edit a dashboard.
	EditableTypes = struct {
		Owner    EditableType
		All      EditableType
		ReadOnly EditableType
	}{
		Owner:    "editable_by_owner",
		All:      "editable_by_all",
		ReadOnly: "read_only",
	}
)

// DashboardIconType represents an option for the dashboard's icon field.
type DashboardIconType string

var (
	// DashboardIconTypes specifies the possible options for dashboard icons.
	DashboardIconTypes = struct {
		Adjust       DashboardIconType
		Archive      DashboardIconType
		BarChart     DashboardIconType
		Bell         DashboardIconType
		Bolt         DashboardIconType
		Bug          DashboardIconType
		Bullhorn     DashboardIconType
		Bullseye     DashboardIconType
		Clock        DashboardIconType
		Cloud        DashboardIconType
		Cog          DashboardIconType
		Comments     DashboardIconType
		Crosshairs   DashboardIconType
		Dashboard    DashboardIconType
		Envelope     DashboardIconType
		Fire         DashboardIconType
		Flag         DashboardIconType
		Flask        DashboardIconType
		Globe        DashboardIconType
		Heart        DashboardIconType
		Leaf         DashboardIconType
		Legal        DashboardIconType
		LifeRing     DashboardIconType
		LineChart    DashboardIconType
		Magic        DashboardIconType
		Mobile       DashboardIconType
		Money        DashboardIconType
		None         DashboardIconType
		PaperPlane   DashboardIconType
		PieChart     DashboardIconType
		PuzzlePiece  DashboardIconType
		Road         DashboardIconType
		Rocket       DashboardIconType
		ShoppingCart DashboardIconType
		Sitemap      DashboardIconType
		Sliders      DashboardIconType
		Tablet       DashboardIconType
		ThumbsDown   DashboardIconType
		ThumbsUp     DashboardIconType
		Trophy       DashboardIconType
		USD          DashboardIconType
		User         DashboardIconType
		Users        DashboardIconType
	}{
		Adjust:       "adjust",
		Archive:      "archive",
		BarChart:     "bar-chart",
		Bell:         "bell",
		Bolt:         "bolt",
		Bug:          "bug",
		Bullhorn:     "bullhorn",
		Bullseye:     "bullseye",
		Clock:        "clock-o",
		Cloud:        "cloud",
		Cog:          "cog",
		Comments:     "comments-o",
		Crosshairs:   "crosshairs",
		Dashboard:    "dashboard",
		Envelope:     "envelope",
		Fire:         "fire",
		Flag:         "flag",
		Flask:        "flask",
		Globe:        "globe",
		Heart:        "heart",
		Leaf:         "leaf",
		Legal:        "legal",
		LifeRing:     "life-ring",
		LineChart:    "line-chart",
		Magic:        "magic",
		Mobile:       "mobile",
		Money:        "money",
		None:         "none",
		PaperPlane:   "paper-plane",
		PieChart:     "pie-chart",
		PuzzlePiece:  "puzzle-piece",
		Road:         "road",
		Rocket:       "rocket",
		ShoppingCart: "shopping-cart",
		Sitemap:      "sitemap",
		Sliders:      "sliders",
		Tablet:       "tablet",
		ThumbsDown:   "thumbs-down",
		ThumbsUp:     "thumbs-up",
		Trophy:       "trophy",
		USD:          "usd",
		User:         "user",
		Users:        "users",
	}
)

// VisualizationType represents an option for adashboard widget's type.
type VisualizationType string

var (
	// VisualizationTypes specifies the possible options for dashboard widget types.
	VisualizationTypes = struct {
		ApplicationBreakdown VisualizationType
		AttributeSheet       VisualizationType
		Billboard            VisualizationType
		BillboardComparison  VisualizationType
		ComparisonLineChart  VisualizationType
		EventFeed            VisualizationType
		EventTable           VisualizationType
		FacetBarChart        VisualizationType
		FacetPieChart        VisualizationType
		FacetTable           VisualizationType
		FacetedAreaChart     VisualizationType
		FacetedLineChart     VisualizationType
		Funnel               VisualizationType
		Gauge                VisualizationType
		Heatmap              VisualizationType
		Histogram            VisualizationType
		LineChart            VisualizationType
		Markdown             VisualizationType
		MetricLineChart      VisualizationType
		RawJSON              VisualizationType
		SingleEvent          VisualizationType
		UniquesList          VisualizationType
	}{
		ApplicationBreakdown: "application_breakdown",
		AttributeSheet:       "attribute_sheet",
		Billboard:            "billboard",
		BillboardComparison:  "billboard_comparison",
		ComparisonLineChart:  "comparison_line_chart",
		EventFeed:            "event_feed",
		EventTable:           "event_table",
		FacetBarChart:        "facet_bar_chart",
		FacetPieChart:        "facet_pie_chart",
		FacetTable:           "facet_table",
		FacetedAreaChart:     "faceted_area_chart",
		FacetedLineChart:     "faceted_line_chart",
		Funnel:               "funnel",
		Gauge:                "gauge",
		Heatmap:              "heatmap",
		Histogram:            "histogram",
		LineChart:            "line_chart",
		Markdown:             "markdown",
		MetricLineChart:      "metric_line_chart",
		RawJSON:              "raw_json",
		SingleEvent:          "single_event",
		UniquesList:          "uniques_list",
	}
)

// DashboardMetadata represents metadata about the dashboard (like version)
type DashboardMetadata struct {
	Version int `json:"version"`
}

// DashboardWidget represents a widget in a dashboard.
type DashboardWidget struct {
	Visualization VisualizationType           `json:"visualization,omitempty"`
	ID            int                         `json:"widget_id,omitempty"`
	AccountID     int                         `json:"account_id,omitempty"`
	Data          []DashboardWidgetData       `json:"data,omitempty"`
	Presentation  DashboardWidgetPresentation `json:"presentation,omitempty"`
	Layout        DashboardWidgetLayout       `json:"layout,omitempty"`
}

// DashboardWidgetData represents the data backing a dashboard widget.
type DashboardWidgetData struct {
	NRQL          string                           `json:"nrql,omitempty"`
	Source        string                           `json:"source,omitempty"`
	Duration      int                              `json:"duration,omitempty"`
	EndTime       int                              `json:"end_time,omitempty"`
	EntityIds     []int                            `json:"entity_ids,omitempty"`
	CompareWith   []DashboardWidgetDataCompareWith `json:"compare_with,omitempty"`
	Metrics       []DashboardWidgetDataMetric      `json:"metrics,omitempty"`
	RawMetricName string                           `json:"raw_metric_name,omitempty"`
	Facet         string                           `json:"facet,omitempty"`
	OrderBy       string                           `json:"order_by,omitempty"`
	Limit         int                              `json:"limit,omitempty"`
}

// DashboardWidgetDataCompareWith represents the compare with configuration of the widget.
type DashboardWidgetDataCompareWith struct {
	OffsetDuration string                                     `json:"offset_duration,omitempty"`
	Presentation   DashboardWidgetDataCompareWithPresentation `json:"presentation,omitempty"`
}

// DashboardWidgetDataCompareWithPresentation represents the compare with presentation configuration of the widget.
type DashboardWidgetDataCompareWithPresentation struct {
	Name  string `json:"name,omitempty"`
	Color string `json:"color,omitempty"`
}

// DashboardWidgetDataMetric represents the metrics data of the widget.
type DashboardWidgetDataMetric struct {
	Name   string   `json:"name,omitempty"`
	Units  string   `json:"units,omitempty"`
	Scope  string   `json:"scope,omitempty"`
	Values []string `json:"values,omitempty"`
}

// DashboardWidgetPresentation represents the visual presentation of a dashboard widget.
type DashboardWidgetPresentation struct {
	Title                string                    `json:"title,omitempty"`
	Notes                string                    `json:"notes,omitempty"`
	DrilldownDashboardID int                       `json:"drilldown_dashboard_id,omitempty"`
	Threshold            *DashboardWidgetThreshold `json:"threshold,omitempty"`
}

// DashboardWidgetThreshold represents the threshold configuration of a dashboard widget.
type DashboardWidgetThreshold struct {
	Red    float64 `json:"red,omitempty"`
	Yellow float64 `json:"yellow,omitempty"`
}

// DashboardWidgetLayout represents the layout of a widget in a dashboard.
type DashboardWidgetLayout struct {
	Width  int `json:"width"`
	Height int `json:"height"`
	Row    int `json:"row"`
	Column int `json:"column"`
}

// DashboardFilter represents the filter in a dashboard.
type DashboardFilter struct {
	EventTypes []string `json:"event_types,omitempty"`
	Attributes []string `json:"attributes,omitempty"`
}

// RawConfiguration represents the configuration for widgets, it's a replacement for configuration field
type RawConfiguration struct {
	// Used by all widgets
	NRQLQueries     []DashboardWidgetNRQLQueryInput  `json:"nrqlQueries,omitempty"`
	PlatformOptions *RawConfigurationPlatformOptions `json:"platformOptions,omitempty"`

	// Used by viz.bullet
	Limit float64 `json:"limit,omitempty"`

	// Used by viz.markdown
	Text string `json:"text,omitempty"`

	// Used by viz.billboard
	Thresholds []DashboardBillboardWidgetThresholdInput `json:"thresholds,omitempty"`
}

// RawConfigurationPlatformOptions represents the platform widget options
type RawConfigurationPlatformOptions struct {
	IgnoreTimeRange bool `json:"ignoreTimeRange,omitempty"`
}
