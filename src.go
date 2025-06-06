package main

import (
	"bufio"
	"fmt"
	f "fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	ssh "golang.org/x/crypto/ssh"
)

type loginReq struct {
	pemFilePath  string
	passwordKey  string
	serverIPAddr string
}

type sudoReq struct {
	client       *ssh.Client
	sudoPassword string
}

type configureMySQL struct {
	database     string
	serverDBName string
	DBPassword   string
}

type MysqlUser struct {
	userName   string
	password   string
	hostIPAddr string
}

func (lq *loginReq) serverSetup() (ssh.Signer, error) {
	keyBytes, err := os.ReadFile(lq.pemFilePath)
	if err != nil {
		return nil, f.Errorf("faild to read file at path %q: %w", lq.pemFilePath, err)
	}

	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err == nil {
		return signer, nil
	}
	if lq.passwordKey != "" {
		signerWithPass, errPass := ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(lq.passwordKey))
		if errPass == nil {
			return signerWithPass, nil
		}
		return nil, f.Errorf("Failed to parse or read the file at path %q: %w", lq.pemFilePath, err)
	}

	return nil, f.Errorf("Failed to parse the key: %w", err)
}

func (lq *loginReq) createClient() (*ssh.ClientConfig, error) {
	signer, err := lq.serverSetup()
	if err != nil {
		return nil, f.Errorf("Failed to create client with file %s: %w", lq.pemFilePath, err)

	}

	config := &ssh.ClientConfig{
		User: "ubuntu",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	return config, nil
}

func (lq *loginReq) createConnection() (*ssh.Client, error) {
	client, err := lq.createClient()
	if err != nil {
		return nil, f.Errorf("failed to create a client: %w", err)
	}
	dialer := net.Dialer{
		Timeout: 10 * time.Second,
	}

	conn, err := dialer.Dial("tcp", lq.serverIPAddr+":22")
	if err != nil {
		return nil, f.Errorf("Failed to connect the server: %w", err)
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, lq.serverIPAddr+":22", client)
	if err != nil {
		return nil, f.Errorf("failed to create a connection to the server: %w", err)
	}
	return ssh.NewClient(c, chans, reqs), nil

}
func (lq *loginReq) remoteCommand(cmd string) error {

	conn, err := lq.createConnection()
	if err != nil {
		return f.Errorf("%w", err)
	}

	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return f.Errorf("failed to create a new session: %w", err)
	}

	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return f.Errorf("failed to create an stdout pipe: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return f.Errorf("failed to create an stderr pipe: %v", err)
	}

	stderrCh := make(chan string)
	stdoutCh := make(chan string)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			stderrCh <- scanner.Text()
		}
		close(stderrCh)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			stdoutCh <- scanner.Text()
		}
		close(stdoutCh)
	}()

	if err := session.Start(cmd); err != nil {
		return f.Errorf("Error related to executing: %w", err)
	}

	for line := range stdoutCh {
		f.Printf("STDOUT: %v \n", line)
	}

	for line := range stderrCh {
		f.Printf("STDERR: %v \n", line)
	}

	if err := session.Wait(); err != nil {
		f.Printf("Error: %v", err)
	}
	wg.Wait()

	return nil
}

func (sq *sudoReq) sudoCmd(command string) (string, error) {
	session, err := sq.client.NewSession()
	if err != nil {
		return "", f.Errorf("failed to create a new session: %w", err)
	}

	defer session.Close()

	session.Stdin = strings.NewReader(sq.sudoPassword + "\n")

	var stdoutBuf, stderrBuf strings.Builder
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	fCmd := f.Sprintf("sudo -S %s", command)

	if err = session.Run(fCmd); err != nil {
		return stderrBuf.String(), fmt.Errorf("failed to execute sudo command inside the server: %v", err)
	}

	return stdoutBuf.String(), nil

}

// changes from localhost to 0.0.0.0
func (lq *loginReq) changeBindAddrToGeneral(sudoPassword string) (string, string, error) {
	conn, err := lq.createConnection()
	if err != nil {
		return "", "", f.Errorf("%w", err)
	}

	configFilePath := "/etc/mysql/mysql.conf.d/mysqld.cnf"
	alterBindAddressCmd := f.Sprintf(
		`sed -i 's/^\s*bind-address\s*=\s*127.0.0.1/bind-address = 0.0.0.0/' %s`,
		configFilePath,
	)
	sq := &sudoReq{
		client:       conn,
		sudoPassword: sudoPassword,
	}
	retrn, err := sq.sudoCmd(alterBindAddressCmd)
	if err != nil {
		return "", "", f.Errorf("%w", err)
	}

	cmd := "systemctl restart mysql"
	rtrnCmd, err := sq.sudoCmd(cmd)
	if err != nil {
		return "", "", f.Errorf("%w", err)
	}

	return retrn, rtrnCmd, nil
}

func (lq *loginReq) findCurrentBindAddr(sudoPassword string) (string, error) {
	conn, err := lq.createConnection()
	if err != nil {
		return "", f.Errorf("Failed to connect the server: %w", err)
	}

	cmd := f.Sprintf("mysql -u root -e \"SHOW VARIABLES LIKE 'bind_address';\"")

	sq := &sudoReq{
		client:       conn,
		sudoPassword: sudoPassword,
	}

	res, err := sq.sudoCmd(cmd)
	if err != nil {
		return res, f.Errorf("%w", err)
	}

	return res, nil
}
func (lq *loginReq) changeBindAddr(sudoPassword, currentBindAddr, newBindAddr string) (string, string, error) {
	conn, err := lq.createConnection()
	if err != nil {
		return "", "", f.Errorf("%w", err)
	}

	configFilePath := "/etc/mysql/mysql.conf.d/mysqld.cnf"
	alterBindAddressCmd := f.Sprintf(
		`sed -i 's/^\s*bind-address\s*=\s*%s/bind-address = %s/' %s`,
		currentBindAddr, newBindAddr, configFilePath,
	)
	sq := &sudoReq{
		client:       conn,
		sudoPassword: sudoPassword,
	}
	retrn, err := sq.sudoCmd(alterBindAddressCmd)
	if err != nil {
		return "", "", f.Errorf("%w", err)
	}

	cmd := "systemctl restart mysql"
	rtrnCmd, err := sq.sudoCmd(cmd)
	if err != nil {
		return "", "", f.Errorf("%w", err)
	}

	return retrn, rtrnCmd, nil
}

func (lq *loginReq) moveFilesToTheServer(filePath string) (string, error) {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return "", f.Errorf("Faild to find the fild: %w", err)
	}

	if err != nil {
		return "", f.Errorf("Faild to find %s path: %w", filePath, err)
	}

	cmd := exec.Command("scp", "-r", "-i", lq.pemFilePath, filePath, lq.serverIPAddr)

	err1 := cmd.Run()
	if err1 != nil {
		return "", f.Errorf("Failed to deploy the file to the server with path %q: %w", filePath, err1)
	}
	return "file deployed to the server", nil
}
func (msc *configureMySQL) moveMysqlDBToTheServer(lq *loginReq) (string, error) {
	outPutFilePath := msc.serverDBName
	outPutFile, err := os.Create(outPutFilePath)
	if err != nil {
		return "", f.Errorf("faild to create the file: %w", err)
	}

	defer outPutFile.Close()

	cmd := exec.Command("mysqldump", "-u", "root", "-p", f.Sprintf("-p%s", msc.DBPassword), msc.database)

	cmd.Stdout = outPutFile

	err = cmd.Run()
	if err != nil {
		return "", f.Errorf("Failed to run the command: %w", err)
	}

	res, err2 := lq.moveFilesToTheServer(outPutFilePath)
	if err2 != nil {
		return "", f.Errorf("%v", err2)
	}

	return res, nil

}
func (mu *MysqlUser) createUsersMysql(sq *sudoReq, InServerDatabase string) (string, error) {
	cmd := f.Sprintf("mysql -u root -e \"create user '%s'@'%s' identified by '%s'; grant all privileges on %s.* to'%s'@'%s'; flush privileges;\"",
		mu.userName,
		mu.hostIPAddr,
		mu.password,
		InServerDatabase,
		mu.userName,
		mu.hostIPAddr,
	)
	result, err := sq.sudoCmd(cmd)
	if err != nil {
		return result, f.Errorf("%w", err)
	}

	return result, nil

}

func (mu *MysqlUser) createBackupUser(sq *sudoReq, InserverDatabase string) (string, error) {
	cmd := fmt.Sprintf(
		"mysql -u root -e \"CREATE USER IF NOT EXISTS '%s'@'%s' IDENTIFIED BY '%s'; "+
			"GRANT SELECT ON %s.* TO '%s'@'%s'; "+
			"GRANT PROCESS, LOCK TABLES, SHOW VIEW ON *.* TO '%s'@'%s'; "+
			"FLUSH PRIVILEGES;\"",
		mu.userName,
		mu.hostIPAddr,
		mu.password,
		InserverDatabase,
		mu.userName,
		mu.hostIPAddr,
		mu.userName,
		mu.hostIPAddr,
	)
	res, err := sq.sudoCmd(cmd)
	if err != nil {
		return res, f.Errorf("Failed to execute SQL command: %w", err)
	}

	return res, nil
}
func (lq *loginReq) createBackup(sudoPassword, dbPassword, dbUser, dbName string) (string, error) {
	conn, err := lq.createConnection()
	if err != nil {
		return "", f.Errorf("Failed to create a connection to the server: %w", err)
	}

	sq := &sudoReq{
		client:       conn,
		sudoPassword: sudoPassword,
	}
	currentTime := time.Now().Format("2006-01-02_15-04-05")
	backupFile := fmt.Sprintf("%s", currentTime)

	cmd := fmt.Sprintf("mysqldump -u %s -p'%s' %s > %s", dbUser, dbPassword, dbName, backupFile)
	res, err1 := sq.sudoCmd(cmd)
	if err1 != nil {
		return res, f.Errorf("Failed to write the command: %w", err1)
	}

	return res, nil

}
