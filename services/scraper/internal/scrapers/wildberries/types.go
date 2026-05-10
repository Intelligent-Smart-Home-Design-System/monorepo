package wildberries

// detail.json
type WBDetailResponse struct {
	Products []struct {
		ID        int     `json:"id"`
		Brand     string  `json:"brand"`
		Name      string  `json:"name"`
		Rating    float64 `json:"rating"`
		Feedbacks int     `json:"feedbacks"`
		Sizes     []struct {
			Price struct {
				Basic   int `json:"basic"`
				Product int `json:"product"`
			} `json:"price"`
		} `json:"sizes"`
	} `json:"products"`
}

// card.json
type WBCardResponse struct {
	ImtID       int    `json:"imt_id"`
	NmID        int    `json:"nm_id"`
	ImtName     string `json:"imt_name"`
	Description string `json:"description"`
	VendorCode  string `json:"vendor_code"`
}