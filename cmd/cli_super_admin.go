package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/daqing/airway/app/modules/identity"
	"github.com/daqing/airway/lib/repo"
	"golang.org/x/sys/unix"
)

// runCLIInitializeSuperAdmin validates arguments and starts interactive super administrator setup.
func runCLIInitializeSuperAdmin(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: airway cli admin:init")
	}
	return initializeSuperAdmin(os.Stdin, os.Stdout)
}

// initializeSuperAdmin reads credentials and creates a super administrator when setup is pending.
func initializeSuperAdmin(in *os.File, out io.Writer) error {
	dsn, err := cliDSN()
	if err != nil {
		return err
	}
	db, err := repo.NewDB(dsn)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer db.Close()

	service := identity.NewService(db, 12*time.Hour)
	initialized, err := service.IsInitialized(context.Background())
	if err != nil {
		return fmt.Errorf("检查超级管理员初始化状态: %w（请先运行数据库迁移）", err)
	}
	if initialized {
		return fmt.Errorf("超级管理员已经初始化")
	}

	prompter := &adminPrompter{in: in, out: out, reader: bufio.NewReader(in)}
	login, err := prompter.readLine("登录名: ")
	if err != nil {
		return err
	}
	email, err := prompter.readLine("邮箱: ")
	if err != nil {
		return err
	}
	password, err := prompter.readSecret("密码: ")
	if err != nil {
		return err
	}
	confirmation, err := prompter.readSecret("确认密码: ")
	if err != nil {
		return err
	}
	if password != confirmation {
		return fmt.Errorf("两次输入的密码不一致")
	}

	admin, err := service.Initialize(context.Background(), login, email, password, "cli", "")
	if errors.Is(err, identity.ErrAlreadyInitialized) {
		return fmt.Errorf("超级管理员已经初始化")
	}
	if err != nil {
		return fmt.Errorf("初始化超级管理员: %w（请先运行数据库迁移）", err)
	}
	_, _ = fmt.Fprintf(out, "超级管理员 %s 初始化成功。\n", admin.Login)
	return nil
}

// adminPrompter reads the interactive terminal input required for super administrator setup.
type adminPrompter struct {
	in     *os.File
	out    io.Writer
	reader *bufio.Reader
}

// readLine displays a prompt and reads one line of plain text.
func (p *adminPrompter) readLine(prompt string) (string, error) {
	if _, err := fmt.Fprint(p.out, prompt); err != nil {
		return "", err
	}
	line, err := p.reader.ReadString('\n')
	if err != nil && !(errors.Is(err, io.EOF) && line != "") {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// readSecret reads a password with terminal echo disabled and supports non-terminal input.
func (p *adminPrompter) readSecret(prompt string) (string, error) {
	if _, err := fmt.Fprint(p.out, prompt); err != nil {
		return "", err
	}
	if !isTerminal(p.in) {
		line, err := p.reader.ReadString('\n')
		if err != nil && !(errors.Is(err, io.EOF) && line != "") {
			return "", err
		}
		return strings.TrimRight(line, "\r\n"), nil
	}

	state, err := unix.IoctlGetTermios(int(p.in.Fd()), ioctlReadTermios)
	if err != nil {
		return "", err
	}
	secretState := *state
	secretState.Lflag &^= unix.ECHO
	if err := unix.IoctlSetTermios(int(p.in.Fd()), ioctlWriteTermios, &secretState); err != nil {
		return "", err
	}
	defer unix.IoctlSetTermios(int(p.in.Fd()), ioctlWriteTermios, state)

	line, readErr := p.reader.ReadString('\n')
	_, _ = fmt.Fprintln(p.out)
	if readErr != nil && !(errors.Is(readErr, io.EOF) && line != "") {
		return "", readErr
	}
	return strings.TrimRight(line, "\r\n"), nil
}
