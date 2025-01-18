package entity

import "time"

type FieldItem struct {
	TableName string     `json:"table_name" field:"table_name" `
	FieldName string     `json:"field_name" field:"field_name" is_input:"true" is_need_in_arg:"true"`
	Value     string     `is_input:"true" is_need_in_arg:"true"`
	ValueUp   string     `json:"value_up" field:"value_up" is_need_in_arg:"true"`
	Type      string     `is_input:"true" is_need_in_arg:"true"`
	Set       []*SetItem `is_input:"true" is_need_in_arg:"true" is_list:"true" pointer:"SetItem"`
}

type SetItem struct {
	Value string `is_input:"true" is_need_in_arg:"true"`
}

type Filter struct {
	Fields          []*FieldItem `is_input:"true" is_list:"true" pointer:"FieldItem" is_need_in_arg:"true"`
	SortByFieldName string       `json:"sort_by_field_name" field:"sort_by_field_name" is_need_in_arg:"true"`
	SortValue       string       `json:"sort_value" field:"sort_value" is_need_in_arg:"true"`
}

type Pagination struct {
	Count int `json:"count" field:"count" is_input:"true" is_need_in_arg:"true"`
	Size  int `json:"size" field:"size" is_input:"true" is_need_in_arg:"true"`
}

type Sorting struct {
	SortByFieldName string `json:"sort_by_field_name" field:"sort_by_field_name" is_need_in_arg:"true"`
	SortValue       string `json:"sort_value" field:"sort_value" is_need_in_arg:"true"`
}

type Period struct {
	Down time.Time `json:"down" field:"down" type:"string" is_need_in_arg:"true"`
	Up   time.Time `json:"up" field:"up" type:"string" is_need_in_arg:"true"`
}
