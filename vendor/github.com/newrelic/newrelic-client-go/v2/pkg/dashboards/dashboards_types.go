package dashboards

import (
	"time"
)

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

	Thresholds        interface{}                       `json:"thresholds,omitempty"`
	Legend            *DashboardWidgetLegend            `json:"legend,omitempty"`
	YAxisLeft         *DashboardWidgetYAxisLeft         `json:"yAxisLeft,omitempty"`
	YAxisRight        *DashboardWidgetYAxisRight        `json:"yAxisRight,omitempty"`
	NullValues        *DashboardWidgetNullValues        `json:"nullValues,omitempty"`
	Units             *DashboardWidgetUnits             `json:"units,omitempty"`
	Colors            *DashboardWidgetColors            `json:"colors,omitempty"`
	Facet             *DashboardWidgetFacet             `json:"facet,omitempty"`
	RefreshRate       *DashboardWidgetRefreshRate       `json:"refreshRate,omitempty"`
	InitialSorting    *DashboardWidgetInitialSorting    `json:"initialSorting,omitempty"`
	DataFormat        []*DashboardWidgetDataFormat      `json:"dataFormatters,omitempty"`
	Tooltip           *DashboardWidgetTooltip           `json:"tooltip,omitempty"`
	BillboardSettings *DashboardWidgetBillboardSettings `json:"billboardSettings,omitempty"`
	ChartStyles       *DashboardWidgetChartStyles       `json:"chartStyles,omitempty"`
}

// RawConfigurationPlatformOptions represents platform widget options
type RawConfigurationPlatformOptions struct {
	IgnoreTimeRange bool `json:"ignoreTimeRange,omitempty"`
}

type DashboardWidgetLegend struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type DashboardWidgetRefreshRate struct {
	Frequency interface{} `json:"frequency,omitempty"`
}

type DashboardWidgetInitialSorting struct {
	Direction string `json:"direction,omitempty"`
	Name      string `json:"name,omitempty"`
}

type DashboardWidgetDataFormat struct {
	Name      string `json:"name,omitempty"`
	Type      string `json:"type,omitempty"`
	Format    string `json:"format,omitempty"`
	Precision int    `json:"precision,omitempty"`
}

type DashboardWidgetYAxisLeft struct {
	Max  float64  `json:"max,omitempty"`
	Min  *float64 `json:"min,omitempty"`
	Zero *bool    `json:"zero,omitempty"`
}

type DashboardWidgetYAxisRight struct {
	Max    float64                           `json:"max,omitempty"`
	Min    *float64                          `json:"min,omitempty"`
	Zero   *bool                             `json:"zero,omitempty"`
	Series []DashboardWidgetYAxisRightSeries `json:"series,omitempty"`
}

type DashboardWidgetYAxisRightSeries struct {
	Name DashboardWidgetYAxisRightSeriesName `json:"name,omitempty"`
}

type DashboardWidgetYAxisRightSeriesName string

type DashboardWidgetNullValues struct {
	NullValue       string                              `json:"nullValue,omitempty"`
	SeriesOverrides []DashboardWidgetNullValueOverrides `json:"seriesOverrides,omitempty"`
}

type DashboardWidgetNullValueOverrides struct {
	NullValue  string `json:"nullValue,omitempty"`
	SeriesName string `json:"seriesName,omitempty"`
}
type DashboardWidgetUnits struct {
	Unit            string                         `json:"unit,omitempty"`
	SeriesOverrides []DashboardWidgetUnitOverrides `json:"seriesOverrides,omitempty"`
}

type DashboardWidgetUnitOverrides struct {
	Unit       string `json:"unit,omitempty"`
	SeriesName string `json:"seriesName"`
}

type DashboardWidgetColors struct {
	Color           string                          `json:"color,omitempty"`
	SeriesOverrides []DashboardWidgetColorOverrides `json:"seriesOverrides,omitempty"`
}

type DashboardWidgetColorOverrides struct {
	Color      string `json:"color,omitempty"`
	SeriesName string `json:"seriesName,omitempty"`
}
type DashboardWidgetFacet struct {
	ShowOtherSeries bool `json:"showOtherSeries,omitempty"`
}

type DashboardWidgetTooltip struct {
	Mode string `json:"mode,omitempty"`
}

type DashboardWidgetChartStyles struct {
	LineInterpolation DashboardLineInterpolationType      `json:"lineInterpolation,omitempty"`
	Gradient          *DashboardWidgetChartStylesGradient `json:"gradient,omitempty"`
}

type DashboardWidgetChartStylesGradient struct {
	Enabled bool `json:"enabled,omitempty"`
}

// DashboardLineInterpolationType represents an option for line chart interpolation.
type DashboardLineInterpolationType string

var DashboardLineInterpolationTypes = struct {
	LINEAR     DashboardLineInterpolationType
	SMOOTH     DashboardLineInterpolationType
	STEPBEFORE DashboardLineInterpolationType
	STEPAFTER  DashboardLineInterpolationType
}{
	LINEAR:     "linear",
	SMOOTH:     "smooth",
	STEPBEFORE: "stepBefore",
	STEPAFTER:  "stepAfter",
}

// DashboardWidgetBillboardSettings represents the billboard settings configuration
type DashboardWidgetBillboardSettings struct {
	Link        *DashboardWidgetBillboardLink        `json:"link,omitempty"`
	Visual      *DashboardWidgetBillboardVisual      `json:"visual,omitempty"`
	GridOptions *DashboardWidgetBillboardGridOptions `json:"gridOptions,omitempty"`
}

// DashboardWidgetBillboardLink represents the link configuration for billboard widgets
type DashboardWidgetBillboardLink struct {
	Title  string `json:"title,omitempty"`
	URL    string `json:"url,omitempty"`
	NewTab bool   `json:"newTab,omitempty"`
}

// DashboardWidgetBillboardVisual represents the visual configuration for billboard widgets
type DashboardWidgetBillboardVisual struct {
	Alignment DashboardBillboardAlignmentType `json:"alignment,omitempty"`
	Display   DashboardBillboardDisplayType   `json:"display,omitempty"`
}

// DashboardBillboardAlignmentType represents an option for the billboard alignment field.
type DashboardBillboardAlignmentType string

var DashboardBillboardAlignmentTypes = struct {
	STACKED DashboardBillboardAlignmentType
	INLINE  DashboardBillboardAlignmentType
}{
	STACKED: "stacked",
	INLINE:  "inline",
}

// DashboardBillboardDisplayType represents an option for the billboard display field.
type DashboardBillboardDisplayType string

var DashboardBillboardDisplayTypes = struct {
	AUTO  DashboardBillboardDisplayType
	ALL   DashboardBillboardDisplayType
	NONE  DashboardBillboardDisplayType
	LABEL DashboardBillboardDisplayType
	VALUE DashboardBillboardDisplayType
}{
	AUTO:  "auto",
	ALL:   "all",
	NONE:  "none",
	LABEL: "label",
	VALUE: "value",
}

// DashboardWidgetBillboardGridOptions represents the grid options for billboard widgets
type DashboardWidgetBillboardGridOptions struct {
	Value   int `json:"value,omitempty"`
	Label   int `json:"label,omitempty"`
	Columns int `json:"columns,omitempty"`
}

// DashboardTooltipType represents an option for the dashboard tooltip's mode field.
type DashboardTooltipType string

var DashboardTooltipTypes = struct {
	ALL    DashboardTooltipType
	SINGLE DashboardTooltipType
	HIDDEN DashboardTooltipType
}{
	ALL:    "all",
	SINGLE: "single",
	HIDDEN: "hidden",
}
