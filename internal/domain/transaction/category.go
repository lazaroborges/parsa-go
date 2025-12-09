package transaction

type TransactionCategory struct {
	ID              int64  `json:"id"`
	OpenFinanceName string `json:"openFinanceName"`
	ParsaName       string `json:"parsaName"`
}

// CategoryMapping maps OpenFinance category codes to TransactionCategory
// Key: OpenFinance category code (e.g., "01000000")
// Value: Category with OpenFinanceName (what API returns) and ParsaName (what mobile expects)
var CategoryMapping = map[string]TransactionCategory{
	"01000000": {
		OpenFinanceName: "Renda",
		ParsaName:       "Renda Ativa",
	},
	"01010000": {
		OpenFinanceName: "Salário",
		ParsaName:       "Salário",
	},
	"01010001": {
		OpenFinanceName: "Pro-labore",
		ParsaName:       "Pro-labore",
	},
	"01019999": {
		OpenFinanceName: "Benefícios",
		ParsaName:       "Benefícios",
	},
	"01020000": {
		OpenFinanceName: "Aposentadoria",
		ParsaName:       "Aposentadoria / Benefício Previdenciário",
	},
	"01030000": {
		OpenFinanceName: "Atividades de empreendedorismo",
		ParsaName:       "Despesas do Trabalho",
	},
	"01040000": {
		OpenFinanceName: "Auxílio do governo",
		ParsaName:       "Auxílio Governamental",
	},
	"01050000": {
		OpenFinanceName: "Renda não-recorrente",
		ParsaName:       "Renda não-recorrente",
	},
	"01999999": {
		OpenFinanceName: "Contas",
		ParsaName:       "Contas",
	},
	"02000000": {
		OpenFinanceName: "Empréstimos e financiamento",
		ParsaName:       "Empréstimos",
	},
	"02010000": {
		OpenFinanceName: "Atraso no pagamento e custos de cheque especial",
		ParsaName:       "Juros / Multas por Atraso",
	},
	"02020000": {
		OpenFinanceName: "Juros cobrados",
		ParsaName:       "Juros / Multas por Atraso",
	},
	"02030000": {
		OpenFinanceName: "Financiamento",
		ParsaName:       "Empréstimos",
	},
	"02030001": {
		OpenFinanceName: "Financiamento imobiliário",
		ParsaName:       "Financiamento imobiliário",
	},
	"02030002": {
		OpenFinanceName: "Financiamento de veículos",
		ParsaName:       "Financiamento de veículos",
	},
	"02030003": {
		OpenFinanceName: "Empréstimo estudantil",
		ParsaName:       "Empréstimos",
	},
	"02040000": {
		OpenFinanceName: "Empréstimos",
		ParsaName:       "Empréstimos",
	},
	"02999998": {
		OpenFinanceName: "Aluguéis",
		ParsaName:       "Aluguéis",
	},
	"02999999": {
		OpenFinanceName: "Venda de Ativos",
		ParsaName:       "Venda de Ativos",
	},
	"03000000": {
		OpenFinanceName: "Investimentos",
		ParsaName:       "Investimentos",
	},
	"03010000": {
		OpenFinanceName: "Investimento automático",
		ParsaName:       "Investimento automático",
	},
	"03020000": {
		OpenFinanceName: "Renda fixa",
		ParsaName:       "Juros e Dividendos",
	},
	"03030000": {
		OpenFinanceName: "Fundos multimercado",
		ParsaName:       "Fundos Multimercados",
	},
	"03040000": {
		OpenFinanceName: "Renda variável",
		ParsaName:       "Renda Variável",
	},
	"03050000": {
		OpenFinanceName: "Ajuste de margem",
		ParsaName:       "Investimentos",
	},
	"03050009": {
		OpenFinanceName: "Renda Passiva",
		ParsaName:       "Renda Passiva",
	},
	"03060000": {
		OpenFinanceName: "Juros de rendimentos de dividendos",
		ParsaName:       "Juros e Dividendos",
	},
	"03060001": {
		OpenFinanceName: "Outros Proventos",
		ParsaName:       "Outros Proventos",
	},
	"03070000": {
		OpenFinanceName: "Pensão",
		ParsaName:       "Pensão",
	},
	"04000000": {
		OpenFinanceName: "Transferência mesma titularidade",
		ParsaName:       "Transferência Bancária",
	},
	"04010000": {
		OpenFinanceName: "Transferência mesma titularidade - Dinheiro",
		ParsaName:       "Transferência Bancária",
	},
	"04020000": {
		OpenFinanceName: "Transferência mesma titularidade - PIX",
		ParsaName:       "Transferência Bancária",
	},
	"04030000": {
		OpenFinanceName: "Transferência mesma titularidade - TED",
		ParsaName:       "Transferência Bancária",
	},
	"05000000": {
		OpenFinanceName: "Transferências",
		ParsaName:       "Transferência Bancária",
	},
	"05010000": {
		OpenFinanceName: "Transferência - Boleto bancário",
		ParsaName:       "Transferência Bancária",
	},
	"05020000": {
		OpenFinanceName: "Transferência - Dinheiro",
		ParsaName:       "Transferência Bancária",
	},
	"05030000": {
		OpenFinanceName: "Transferência - Cheque",
		ParsaName:       "Transferência Bancária",
	},
	"05040000": {
		OpenFinanceName: "Transferências- DOC",
		ParsaName:       "Transferência Bancária",
	},
	"05050000": {
		OpenFinanceName: "Transferência - Câmbio",
		ParsaName:       "Transferência Bancária",
	},
	"05060000": {
		OpenFinanceName: "Transferência - Mesma instituição",
		ParsaName:       "Transferência Bancária",
	},
	"05070000": {
		OpenFinanceName: "Transferência - PIX",
		ParsaName:       "Transferência Bancária",
	},
	"05080000": {
		OpenFinanceName: "Transferência - TED",
		ParsaName:       "Transferência Bancária",
	},
	"05090000": {
		OpenFinanceName: "Transferências para terceiros",
		ParsaName:       "Transferência Bancária",
	},
	"05090001": {
		OpenFinanceName: "Transferência para terceiros - Boleto bancário",
		ParsaName:       "Transferência Bancária",
	},
	"05090002": {
		OpenFinanceName: "Transferência para terceiros - Débito",
		ParsaName:       "Transferência Bancária",
	},
	"05090003": {
		OpenFinanceName: "Transferência para terceiros - DOC",
		ParsaName:       "Transferência Bancária",
	},
	"05090004": {
		OpenFinanceName: "Transferência para terceiros - PIX",
		ParsaName:       "Transferência Bancária",
	},
	"05090005": {
		OpenFinanceName: "Transferência para terceiros - TED",
		ParsaName:       "Transferência Bancária",
	},
	"05100000": {
		OpenFinanceName: "Pagamento de cartão de crédito",
		ParsaName:       "Pagamento Fatura do Cartão",
	},
	"06000000": {
		OpenFinanceName: "Obrigações legais",
		ParsaName:       "Obrigações legais",
	},
	"06010000": {
		OpenFinanceName: "Saldo bloqueado",
		ParsaName:       "Saldo bloqueado",
	},
	"06020000": {
		OpenFinanceName: "Pensão alimentícia",
		ParsaName:       "Pensão alimentícia",
	},
	"07000000": {
		OpenFinanceName: "Serviços",
		ParsaName:       "Serviços",
	},
	"07010000": {
		OpenFinanceName: "Telecomunicação",
		ParsaName:       "Serviços de Telecomunicação",
	},
	"07010001": {
		OpenFinanceName: "Internet",
		ParsaName:       "Serviços de Telecomunicação",
	},
	"07010002": {
		OpenFinanceName: "Celular",
		ParsaName:       "Serviços de Telecomunicação",
	},
	"07010003": {
		OpenFinanceName: "TV",
		ParsaName:       "Serviços de Telecomunicação",
	},
	"07010004": {
		OpenFinanceName: "Serviços de Telecom",
		ParsaName:       "Serviços de Telecom",
	},
	"07020000": {
		OpenFinanceName: "Educação",
		ParsaName:       "Educação",
	},
	"07020001": {
		OpenFinanceName: "Cursos online",
		ParsaName:       "Cursos e Treinamentos",
	},
	"07020002": {
		OpenFinanceName: "Universidade",
		ParsaName:       "Universidade",
	},
	"07020003": {
		OpenFinanceName: "Escola",
		ParsaName:       "Escola",
	},
	"07020004": {
		OpenFinanceName: "Creche",
		ParsaName:       "Creche",
	},
	"07030000": {
		OpenFinanceName: "Saúde e bem-estar",
		ParsaName:       "Saúde e Bem-estar",
	},
	"07030001": {
		OpenFinanceName: "Academia e centros de lazer",
		ParsaName:       "Saúde e Bem-estar",
	},
	"07030002": {
		OpenFinanceName: "Prática de esportes",
		ParsaName:       "Saúde e Bem-estar",
	},
	"07030003": {
		OpenFinanceName: "Bem-estar",
		ParsaName:       "Bem-estar",
	},
	"07040000": {
		OpenFinanceName: "Bilhetes",
		ParsaName:       "Eventos e Cultura",
	},
	"07040001": {
		OpenFinanceName: "Estádios e arenas",
		ParsaName:       "Eventos e Cultura",
	},
	"07040002": {
		OpenFinanceName: "Museus e pontos turísticos",
		ParsaName:       "Eventos e Cultura",
	},
	"07040003": {
		OpenFinanceName: "Cinema, Teatro e Concertos",
		ParsaName:       "Eventos e Cultura",
	},
	"08000000": {
		OpenFinanceName: "Compras",
		ParsaName:       "Compras",
	},
	"08010000": {
		OpenFinanceName: "Compras online",
		ParsaName:       "Compras online",
	},
	"08020000": {
		OpenFinanceName: "Eletrônicos",
		ParsaName:       "Eletrônicos",
	},
	"08030000": {
		OpenFinanceName: "Pet Shops e veterinários",
		ParsaName:       "Pet Shops e Veterinários",
	},
	"08040000": {
		OpenFinanceName: "Vestiário",
		ParsaName:       "Roupas",
	},
	"08050000": {
		OpenFinanceName: "Artigos infantis",
		ParsaName:       "Artigos infantis",
	},
	"08050001": {
		OpenFinanceName: "Higine, Beleza e Perfumaria",
		ParsaName:       "Higine, Beleza e Perfumaria",
	},
	"08060000": {
		OpenFinanceName: "Livraria",
		ParsaName:       "Livraria",
	},
	"08070000": {
		OpenFinanceName: "Artigos esportivos",
		ParsaName:       "Artigos esportivos",
	},
	"08080000": {
		OpenFinanceName: "Papelaria",
		ParsaName:       "Papelaria",
	},
	"08090000": {
		OpenFinanceName: "Cashback",
		ParsaName:       "Cashback",
	},
	"08090001": {
		OpenFinanceName: "Presentes",
		ParsaName:       "Presentes",
	},
	"09000000": {
		OpenFinanceName: "Serviços digitais",
		ParsaName:       "Assinaturas Digitais",
	},
	"09010000": {
		OpenFinanceName: "Jogos e videogames",
		ParsaName:       "Assinaturas Digitais",
	},
	"09020000": {
		OpenFinanceName: "Streaming de vídeo",
		ParsaName:       "Assinaturas Digitais",
	},
	"09030000": {
		OpenFinanceName: "Streaming de música",
		ParsaName:       "Assinaturas Digitais",
	},
	"10000000": {
		OpenFinanceName: "Supermercado",
		ParsaName:       "Mercado",
	},
	"11000000": {
		OpenFinanceName: "Alimentos e bebidas",
		ParsaName:       "Alimentação",
	},
	"11010000": {
		OpenFinanceName: "Restaurantes, bares e lanchonetes",
		ParsaName:       "Restaurantes e Bares",
	},
	"11020000": {
		OpenFinanceName: "Delivery de alimentos",
		ParsaName:       "Delivery",
	},
	"12000000": {
		OpenFinanceName: "Viagens",
		ParsaName:       "Viagens",
	},
	"12010000": {
		OpenFinanceName: "Aeroportos e cias. aéreas",
		ParsaName:       "Transporte Áereo",
	},
	"12020000": {
		OpenFinanceName: "Hospedagem",
		ParsaName:       "Outros",
	},
	"12030000": {
		OpenFinanceName: "Programas de milhagem",
		ParsaName:       "Outros",
	},
	"12040000": {
		OpenFinanceName: "Passagem de ônibus",
		ParsaName:       "Transporte Público",
	},
	"12050000": {
		OpenFinanceName: "Combustível",
		ParsaName:       "Combustível",
	},
	"13000000": {
		OpenFinanceName: "Doações",
		ParsaName:       "Outros",
	},
	"14000000": {
		OpenFinanceName: "Apostas",
		ParsaName:       "Lazer",
	},
	"14010000": {
		OpenFinanceName: "Loteria",
		ParsaName:       "Loteria e Apostas online",
	},
	"14020000": {
		OpenFinanceName: "Apostas online",
		ParsaName:       "Loteria e Apostas online",
	},
	"14030000": {
		OpenFinanceName: "Vida Noturna",
		ParsaName:       "Vida Noturna",
	},
	"15000000": {
		OpenFinanceName: "Impostos",
		ParsaName:       "Impostos",
	},
	"15010000": {
		OpenFinanceName: "Imposto de renda",
		ParsaName:       "Imposto de renda",
	},
	"15020000": {
		OpenFinanceName: "Imposto sobre investimentos",
		ParsaName:       "Impostos",
	},
	"15030000": {
		OpenFinanceName: "Impostos sobre operações financeiras",
		ParsaName:       "Impostos",
	},
	"16000000": {
		OpenFinanceName: "Taxas bancárias",
		ParsaName:       "Taxas Bancárias",
	},
	"16010000": {
		OpenFinanceName: "Taxas de conta corrente",
		ParsaName:       "Taxas Bancárias",
	},
	"16020000": {
		OpenFinanceName: "Taxas sobre transferências e caixa eletrônico",
		ParsaName:       "Taxas Bancárias",
	},
	"16030000": {
		OpenFinanceName: "Taxas de cartão de crédito",
		ParsaName:       "Taxas Bancárias",
	},
	"17000000": {
		OpenFinanceName: "Moradia",
		ParsaName:       "Moradia",
	},
	"17000001": {
		OpenFinanceName: "Serviços Domésticos",
		ParsaName:       "Serviços Domésticos",
	},
	"17010000": {
		OpenFinanceName: "Aluguel",
		ParsaName:       "Aluguel",
	},
	"17020000": {
		OpenFinanceName: "Serviços de utilidade pública",
		ParsaName:       "Serviços de utilidade pública",
	},
	"17020001": {
		OpenFinanceName: "Água",
		ParsaName:       "Água",
	},
	"17020002": {
		OpenFinanceName: "Eletricidade",
		ParsaName:       "Energia elétrica",
	},
	"17020003": {
		OpenFinanceName: "Gás",
		ParsaName:       "Gás",
	},
	"17030000": {
		OpenFinanceName: "Utensílios para casa",
		ParsaName:       "Produtos para o lar",
	},
	"17040000": {
		OpenFinanceName: "Impostos sobre moradia",
		ParsaName:       "Impostos Residenciais",
	},
	"17050000": {
		OpenFinanceName: "Móveis e eletrodomésticos",
		ParsaName:       "Móveis e eletrodomésticos",
	},
	"18000000": {
		OpenFinanceName: "Saúde",
		ParsaName:       "Saúde e Bem-estar",
	},
	"18010000": {
		OpenFinanceName: "Dentista",
		ParsaName:       "Profissional da Saúde",
	},
	"18020000": {
		OpenFinanceName: "Farmácia",
		ParsaName:       "Farmácia",
	},
	"18030000": {
		OpenFinanceName: "Ótica",
		ParsaName:       "Saúde e Bem-estar",
	},
	"18040000": {
		OpenFinanceName: "Hospitais, clínicas e laboratórios",
		ParsaName:       "Hospitais, clínicas e laboratórios",
	},
	"18050000": {
		OpenFinanceName: "Esportes",
		ParsaName:       "Esportes",
	},
	"19000000": {
		OpenFinanceName: "Transporte",
		ParsaName:       "Transporte",
	},
	"19010000": {
		OpenFinanceName: "Táxi e transporte privado urbano",
		ParsaName:       "Táxi e transporte privado urbano",
	},
	"19020000": {
		OpenFinanceName: "Transporte público",
		ParsaName:       "Transporte Público",
	},
	"19030000": {
		OpenFinanceName: "Aluguel de veículos",
		ParsaName:       "Despesas Veículo Particular",
	},
	"19040000": {
		OpenFinanceName: "Aluguel de bicicletas",
		ParsaName:       "Despesas Veículo Particular",
	},
	"19050000": {
		OpenFinanceName: "Serviços automotivos",
		ParsaName:       "Despesas Veículo Particular",
	},
	"19050001": {
		OpenFinanceName: "Postos de gasolina",
		ParsaName:       "Despesas Veículo Particular",
	},
	"19050002": {
		OpenFinanceName: "Estacionamentos",
		ParsaName:       "Despesas Veículo Particular",
	},
	"19050003": {
		OpenFinanceName: "Pedágios e pagamentos no veículo",
		ParsaName:       "Despesas Veículo Particular",
	},
	"19050004": {
		OpenFinanceName: "Taxas e impostos sobre veículos",
		ParsaName:       "Despesas Veículo Particular",
	},
	"19050005": {
		OpenFinanceName: "Manutenção de veículos",
		ParsaName:       "Despesas Veículo Particular",
	},
	"19050006": {
		OpenFinanceName: "Multas de trânsito",
		ParsaName:       "Despesas Veículo Particular",
	},
	"20000000": {
		OpenFinanceName: "Seguros",
		ParsaName:       "Seguros",
	},
	"200100000": {
		OpenFinanceName: "Seguro de vida",
		ParsaName:       "Seguros",
	},
	"200200000": {
		OpenFinanceName: "Seguro residencial",
		ParsaName:       "Seguros",
	},
	"200300000": {
		OpenFinanceName: "Seguro saúde",
		ParsaName:       "Seguros",
	},
	"200400000": {
		OpenFinanceName: "Seguro de veículos",
		ParsaName:       "Seguros",
	},
	"21000000": {
		OpenFinanceName: "Lazer",
		ParsaName:       "Lazer",
	},
	"99999996": {
		OpenFinanceName: "Doações e Contribuições",
		ParsaName:       "Doações e Contribuições",
	},
	"99999997": {
		OpenFinanceName: "Ajuda Familiar",
		ParsaName:       "Ajuda Familiar",
	},
	"99999998": {
		OpenFinanceName: "Despesa Não Classificada",
		ParsaName:       "Despesa Não Classificada",
	},
	"99999999": {
		OpenFinanceName: "Outros",
		ParsaName:       "Outros",
	},
}

// GetCategoryKey returns the category code (Key) from OpenFinanceName or code
// If category is already a code (8 digits), returns it as-is
// If category is an OpenFinanceName, performs reverse lookup to find the Key
// Returns nil if no mapping is found
func GetCategoryKey(category *string) *string {
	if category == nil || *category == "" {
		return nil
	}

	// If it's already a code (exists as key in mapping), return it
	if _, ok := CategoryMapping[*category]; ok {
		return category
	}

	// Search by OpenFinanceName to find the Key
	for key, cat := range CategoryMapping {
		if cat.OpenFinanceName == *category {
			return &key
		}
	}

	// No mapping found
	return nil
}

// TranslateCategory translates an OpenFinance category (code or name) to ParsaName
// It handles two cases:
// 1. If category is a code (e.g., "01000000"), looks it up directly in CategoryMapping
// 2. If category is a name (e.g., "Renda"), searches by OpenFinanceName
// Returns the ParsaName if found, otherwise returns the original category as fallback
func TranslateCategory(category *string) *string {
	if category == nil || *category == "" {
		return nil
	}

	// First, try direct lookup by code
	if cat, ok := CategoryMapping[*category]; ok {
		parsaName := cat.ParsaName
		return &parsaName
	}

	// If not found by code, search by OpenFinanceName
	for _, cat := range CategoryMapping {
		if cat.OpenFinanceName == *category {
			parsaName := cat.ParsaName
			return &parsaName
		}
	}

	// Fallback: return original category if no mapping found
	return category
}
