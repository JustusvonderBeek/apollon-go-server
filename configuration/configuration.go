package configuration

type Config struct {
	Secure             bool
	ListenAddr         string
	ListenPort         string
	SecureListenPort   string
	RestApiPort        string
	RestApi            bool
	Logfile            string
	ClearDatabase      bool
	CertificateFile    string
	CertificateKeyfile string
	DatabaseFile       string
	DatabaseNoWrite    bool
}
