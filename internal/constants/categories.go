package constants

type Category struct {
	Id          string
	Description string
}

var BNECategories = []Category{
	{Id: "GRAFNOPRO", Description: "Dibujos, carteles, efímera, grabados, fotografías"},
	{Id: "GRAFPRO", Description: "Filminas, transparencias"},
	{Id: "GRABSONORA", Description: "Grabaciones sonoras"},
	{Id: "KIT", Description: "Kit o multimedia"},
	{Id: "MANUSCRITO", Description: "Manuscritos y archivos personales"},
	{Id: "CARTOGRAFI", Description: "Mapas"},
	{Id: "MATEMIXTO", Description: "Materiales mixtos"},
	{Id: "MONOANTIGU", Description: "Monografías antiguas"},
	{Id: "MONOMODERN", Description: "Monografías modernas"},
	{Id: "MUSICAESC", Description: "Partituras"},
	{Id: "RECELECTRO", Description: "Recursos electrónicos"},
	{Id: "SERIADA", Description: "Prensa y revistas"},
	{Id: "VIDEO", Description: "Videograbaciones"},
}

const (
	BaseURL       = "https://www.bne.es/redBNE/alma/SuministroRegistros/Bibliograficos"
	MRCFileSuffix = "-mrc_new.mrc"
)
