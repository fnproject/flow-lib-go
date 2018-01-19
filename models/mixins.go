package models

func (d *ModelDatum) InnerDatum() interface{} {
	if d.Blob != nil {
		return d.Blob
	} else if d.Error != nil {
		return d.Error
	} else if d.HTTPReq != nil {
		return d.HTTPReq
	} else if d.HTTPResp != nil {
		return d.HTTPResp
	} else if d.StageRef != nil {
		return d.StageRef
	} else if d.Status != nil {
		return d.Status
	} else if d.Empty != nil {
		return d.Empty
	}
	panic("Datum is empty!")
}
