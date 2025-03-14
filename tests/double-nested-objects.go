type AutoGenerated struct {
	First  First  `json:"first"`
	Second Second `json:"second"`
}
type Type struct {
	Short string `json:"short"`
	Long  string `json:"long"`
}
type First struct {
	ID   int  `json:"id"`
	Type Type `json:"type"`
}
type SecondType struct {
	Long string `json:"long"`
}
type Second struct {
	ID         int        `json:"id"`
	SecondType SecondType `json:"type"`
}
