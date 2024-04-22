You only need to import the database, modify the config.json, and run it directly

The project has been deployed at https://blob20.art

API:
	GET /api/getAccounts
		params：
			page 		int	default value: 1
			pageSize	int	default value: 100
			address		string
			protocol	string
			ticker 		string

	GET /api/getTickers
		params:
			tx			string
			protocol	string
			ticker 		string

	GET /api/getRecords
		params:
			page 		int	default value: 1
			pageSize	int	default value: 20
			tx			string
			from		string
			to 			string
			protocol	string
			ticker		string
			operation	string
