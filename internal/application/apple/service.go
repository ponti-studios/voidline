package apple

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"voidline/internal/domain/apple"
	"voidline/internal/infrastructure/persistence/sqlite"
)

type Service struct {
	repo *sqlite.AppleRepository
}

func NewService(repo *sqlite.AppleRepository) *Service {
	return &Service{repo: repo}
}

type ImportOptions struct {
	DryRun bool
	Force  bool
}

func (s *Service) ImportAll(ctx context.Context, sourceDir string, options ImportOptions) (*apple.ImportResult, error) {
	result := &apple.ImportResult{}

	if err := s.repo.EnsureTables(ctx); err != nil {
		return result, fmt.Errorf("failed to ensure tables: %w", err)
	}

	contactsResult, err := s.ImportContacts(ctx, sourceDir, options)
	if err != nil {
		fmt.Printf("Contacts import error: %v\n", err)
	}

	notesResult, err := s.ImportNotes(ctx, sourceDir, options)
	if err != nil {
		fmt.Printf("Notes import error: %v\n", err)
	}

	result.TotalRows = contactsResult.TotalRows + notesResult.TotalRows
	result.Inserted = contactsResult.Inserted + notesResult.Inserted
	result.Skipped = contactsResult.Skipped + notesResult.Skipped

	fmt.Println("Apple Personal Data migration complete.")
	return result, nil
}

func (s *Service) ImportContacts(ctx context.Context, sourceDir string, options ImportOptions) (*apple.ImportResult, error) {
	baseDir := sourceDir + "/apple/iCloud Contacts/vCards"
	result := &apple.ImportResult{}

	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		fmt.Printf("Skipping Contacts: %s not found.\n", baseDir)
		return result, nil
	}

	fmt.Println("Importing Apple Contacts...")
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return result, fmt.Errorf("failed to read directory: %w", err)
	}

	count := 0
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".vcf") {
			continue
		}

		path := filepath.Join(baseDir, entry.Name())
		data, err := parseVCard(path)
		if err != nil {
			fmt.Printf("Error parsing %s: %v\n", entry.Name(), err)
			continue
		}

		if data.Name == "" {
			continue
		}

		if !options.DryRun {
			if err := s.repo.InsertContact(ctx, data); err != nil {
				fmt.Printf("Error inserting %s: %v\n", entry.Name(), err)
				continue
			}
		}
		count++
	}

	result.Inserted = count
	fmt.Printf("Imported %d contacts.\n", count)
	return result, nil
}

func (s *Service) ImportNotes(ctx context.Context, sourceDir string, options ImportOptions) (*apple.ImportResult, error) {
	baseDir := sourceDir + "/apple/iCloud Notes/Notes"
	result := &apple.ImportResult{}

	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		fmt.Printf("Skipping Notes: %s not found.\n", baseDir)
		return result, nil
	}

	fmt.Println("Importing Apple Notes...")
	count := 0

	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".txt") {
			return nil
		}

		folderName := filepath.Base(filepath.Dir(path))
		if folderName == "Notes" {
			folderName = "Root"
		}

		file, err := os.Open(path)
		if err != nil {
			fmt.Printf("Error opening %s: %v\n", path, err)
			return nil
		}
		defer file.Close()

		content, err := bufio.NewReader(file).ReadString('\n')
		if err != nil && err.Error() != "EOF" {
			fmt.Printf("Error reading %s: %v\n", path, err)
			return nil
		}

		title := strings.TrimSuffix(info.Name(), ".txt")
		tsMatch := regexp.MustCompile(`(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z)`).FindStringSubmatch(info.Name())
		createdAt := ""
		if len(tsMatch) > 1 {
			createdAt = tsMatch[1]
		}

		note := &apple.Note{
			Title:      title,
			Content:    content,
			Folder:     folderName,
			SourceFile: info.Name(),
			CreatedAt:  createdAt,
		}

		if !options.DryRun {
			if err := s.repo.InsertNote(ctx, note); err != nil {
				fmt.Printf("Error inserting note %s: %v\n", info.Name(), err)
				return nil
			}
		}
		count++
		return nil
	})

	if err != nil {
		return result, fmt.Errorf("error walking directory: %w", err)
	}

	result.Inserted = count
	fmt.Printf("Imported %d notes.\n", count)
	return result, nil
}

func parseVCard(path string) (*apple.Contact, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	contact := &apple.Contact{}

	nameMatch := regexp.MustCompile(`FN:(.*)`).FindStringSubmatch(content)
	if len(nameMatch) > 1 {
		contact.Name = strings.TrimSpace(nameMatch[1])
	}

	phoneMatch := regexp.MustCompile(`TEL.*:(.*)`).FindStringSubmatch(content)
	if len(phoneMatch) > 1 {
		contact.Phone = strings.TrimSpace(phoneMatch[1])
	}

	emailMatch := regexp.MustCompile(`EMAIL.*:(.*)`).FindStringSubmatch(content)
	if len(emailMatch) > 1 {
		contact.Email = strings.TrimSpace(emailMatch[1])
	}

	orgMatch := regexp.MustCompile(`ORG:(.*)`).FindStringSubmatch(content)
	if len(orgMatch) > 1 {
		contact.Organization = strings.TrimSpace(orgMatch[1])
	}

	return contact, nil
}
