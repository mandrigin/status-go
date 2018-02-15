package chanreader


func main() {
	cwd, _ := os.Getwd()
	db, err := leveldb.OpenFile(cwd+"/data", nil)
	if err != nil {
		log.Fatal("can't open levelDB file. ERR: %v", err)
	}
	defer db.Close()

	config, err := bots.NodeConfig()
	if err != nil {
		log.Fatalf("Making config failed: %v", err)
		return
	}

	backend := api.NewStatusBackend()
	log.Println("Starting node...")
	started, err := backend.StartNode(config)
	if err != nil {
		log.Fatalf("Node start failed: %v", err)
		return
	}

	node, err := backend.NodeManager().Node()
	if err != nil {
		log.Fatalf("Getting node failed: %v", err)
		return
	}

	bots
	.SignupOrLogin(node, "my-cool-password")
	.Join("humans-need-not-apply", "Cloudy Test Baboon")
	.RepeatEvery(10 * time.Second, func(ch *StatusChannel) {
		message := fmt.Sprintf("Gopher, gopher: %d", time.Now().Unix())
		ch.WriteMessage(message)
	})

	//loginAndRun(api.NewStatusAPIWithBackend(backend), db)

	// wait till node has been stopped
	node.Wait()
}
