package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/jacobsa/go-serial/serial"
)

var (
	comPort  string
	baudRate int
	delay    int
	command  string
	cases    map[string][]string
	result   map[string][]string
	commands []string
	hreports []Hreport
	filename string
	bserver  bool
	f        *os.File
)

type Hreport struct {
	Command string
	Exp     string
	Get     string
	Result  string
}

func init() {
	cases = make(map[string][]string)
	result = make(map[string][]string)
	commands = make([]string, 0, 500)
}

func ReadCase() {
	fileHandle, err := os.Open("design.md")
	if err != nil {
		log.Fatal("read case file error")
	}
	defer fileHandle.Close()

	fileScanner := bufio.NewScanner(fileHandle)

	for fileScanner.Scan() {
		line := fileScanner.Text()
		if strings.Contains(line, "#") {
			getConfig(line)
		} else {
			makeUpCase(line)
		}
	}

	fmt.Printf("%s %d %d\n", comPort, baudRate, delay)
	//fmt.Printf("%+v\n", cases)
}

func getConfig(cfg string) {
	res := strings.Split(strings.TrimLeft(cfg, "#"), "=")
	if strings.Compare(res[0], "port") == 0 {
		comPort = res[1]
	} else if strings.Compare(res[0], "baudrate") == 0 {
		baudRate, _ = strconv.Atoi(res[1])
	} else if strings.Compare(res[0], "sleep") == 0 {
		delay, _ = strconv.Atoi(res[1])
	}
}

func makeUpCase(line string) {
	if strings.Contains(line, "read") {
		res := strings.Split(line, "[")
		expect := strings.Split(strings.TrimRight(res[1], "]"), ",")
		if _, ok := cases[command]; !ok {
			cases[command] = expect
		} else {
			command = command + "#rep"
			cases[command] = expect
		}
		commands = append(commands, command)
	} else if strings.Contains(line, "send") {
		res := strings.Split(line, " ")
		command = strings.TrimSpace(res[1])
	}
}

func RunCase() {
	options := serial.OpenOptions{
		PortName:        comPort,
		BaudRate:        uint(baudRate),
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 2,
	}

	var commandTemp string

	// Open the port.
	port, err := serial.Open(options)
	if err != nil {
		log.Fatalf("serial.Open: %v", err)
	}

	// Make sure to close it later.
	defer port.Close()

	for command := range cases {
		index := strings.IndexByte(command, '#')
		if index != -1 {
			commandTemp = command[:index]
		}
		fmt.Printf("Run %s...\n", commandTemp)
		runCommand(port, commandTemp+"\r", command)
		if delay == 0 {
			delay = 3
		}
		time.Sleep(time.Duration(delay) * time.Second)
	}
}

func runCommand(s io.ReadWriteCloser, command, rawcmd string) {
	_, err := s.Write([]byte(command))
	if err != nil {
		fmt.Printf("%s run error", command)
	}

	buf := make([]byte, 512)
	n, err := s.Read(buf)
	if err != nil {
		fmt.Printf("%s read error", command)
	}

	//fmt.Printf("Read: %+q", buf[:n])
	res := cleanRes(buf[:n])

	result[rawcmd] = res
	//fmt.Printf("command: %+q \t res: %+q\n", command, res)
}

func cleanRes(buf []byte) []string {
	res := strings.Split(string(buf), "\r\n")
	ret := make([]string, 0, len(res))
	for _, value := range res[1:] {
		if value != "" {
			ret = append(ret, value)
		}
	}
	return ret
}

func Report() {

	var i int
	hreports = make([]Hreport, len(commands))

	for _, cmd := range commands {
		expect := cases[cmd]
		value := result[cmd]
		index := strings.IndexByte(cmd, '#')
		if index != -1 {
			cmd = cmd[:index]
		}
		if compare(expect, value) {
			hreports[i].Result = "PASS"
			//fmt.Printf("[PASS]\t%s\n", cmd)
		} else {
			hreports[i].Result = "FAIL"
			//fmt.Printf("[FAIL]\t%s\tExpect:%+q\tGet:%+q\n", cmd, expect, value)
		}
		hreports[i].Command = cmd
		hreports[i].Exp = "[ " + strings.Join(expect, ", ") + " ]"
		hreports[i].Get = "[ " + strings.Join(value, ", ") + " ]"
		i++
	}

	savefile()
}

func currentTime() string {
	t := time.Now()
	return fmt.Sprintf("%d-%02d-%02d-%02d-%02d-%02d",
		t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}

func savefile() {
	date := currentTime()
	filename, _ = filepath.Abs("./report/result_" + date + ".html")
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		fmt.Println("open file error", err)
	}
	defer f.Close()

	t, _ := template.ParseFiles("report.html")
	t.Execute(f, hreports)
}

func checkFileIsExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

func compare(exp, res []string) bool {
	if (exp == nil) != (res == nil) {
		return false
	}

	if len(exp) != len(res) {
		return false
	}

	for i := range exp {
		if strings.TrimSpace(exp[i]) != res[i] {
			return false
		}
	}

	return true
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir("./report")
	if err != nil {
		fmt.Println("read dir error ", err)
	}

	fileSlice := make([]string, 0, len(files))
	for _, file := range files {
		fileSlice = append(fileSlice, file.Name())
	}
	t, _ := template.ParseFiles("index.html")

	t.Execute(w, fileSlice)
}

func RunServer() {
	fs := http.FileServer(http.Dir("report"))
	http.Handle("/report/", http.StripPrefix("/report/", fs))
	http.HandleFunc("/", serveIndex)
	error := http.ListenAndServe(":9000", nil)
	if error != nil {
		panic(error)
	}
}

func main() {
	flag.BoolVar(&bserver, "server", false, "option for view report by server")
	flag.Parse()
	ReadCase()
	fmt.Printf("== Start Test ==\n")
	RunCase()
	fmt.Printf("\n== Start Generate Report... == \n")
	Report()
	if bserver {
		fmt.Println("Please visit localhost:9000 see reports")
		RunServer()
	} else {
		fmt.Printf("Report generated, check %s for more detail\n", filename)
	}

	in := bufio.NewReader(os.Stdin)
	c, _ := in.ReadByte()
	fmt.Println(string(c))
}
