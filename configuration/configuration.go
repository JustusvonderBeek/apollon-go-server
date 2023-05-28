package configuration

type Config struct {
	Secure             bool
	ListenAddr         string
	ListenPort         string
	SecureListenPort   string
	Logfile            string
	ClearDatabase      bool
	CertificateFile    string
	CertificateKeyfile string
	DatabaseFile       string
	DatabaseNoWrite    bool
}
