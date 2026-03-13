package controllers

import (
	"bytes"
	"journal/internal/models"
	"journal/internal/utils"
	"journal/internal/web/blog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/adrg/frontmatter"
	"github.com/gofiber/fiber/v2"
	_ "github.com/joho/godotenv/autoload"
	"github.com/yuin/goldmark"
	"gorm.io/gorm"
)

func init() {
	registerController(&BlogController{})
}

type BlogController struct {
	db    *gorm.DB
	api   fiber.Router
	views fiber.Router
}

func (c *BlogController) Init(db *gorm.DB, app *fiber.App) {
	c.db = db
	c.views = app.Group("blog")
	c.api = app.Group("blog/api")
}

func (c *BlogController) RegisterApiRoutes() {
}

func (c *BlogController) RegisterViewRoutes() {
	c.views.Get("/", c.getBlogs, utils.RenderPage(blog.List))
	c.views.Get("/:slug", c.getBlog, utils.RenderPage(blog.View))
}

func (c *BlogController) getBlogPath() string {
	return filepath.Join("internal", "blog", "entries")
}

func (c *BlogController) getBlogs(ctx *fiber.Ctx) error {
	entries, err := os.ReadDir(c.getBlogPath())
	if err != nil {
		return err
	}

	var blogs []*models.Blog
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}

		content, err := os.ReadFile(filepath.Join(c.getBlogPath(), entry.Name()))
		if err != nil {
			continue
		}

		var blog models.Blog
		_, err = frontmatter.Parse(bytes.NewReader(content), &blog)
		if err != nil {
			continue
		}

		if blog.Date == nil || blog.Date.After(time.Now()) {
			continue
		}

		blog.Slug = strings.TrimSuffix(entry.Name(), ".md")

		blogs = append(blogs, &blog)
	}

	sort.Slice(blogs, func(i, j int) bool {
		return blogs[i].Date.After(*blogs[j].Date)
	})

	ctx.Locals("blogs", &blogs)
	return ctx.Next()
}

func (c *BlogController) getBlog(ctx *fiber.Ctx) error {
	slug := ctx.Params("slug")
	content, err := os.ReadFile(filepath.Join(c.getBlogPath(), slug+".md"))
	if err != nil {
		return ctx.SendStatus(http.StatusNotFound) // err
	}

	var blog models.Blog
	rest, err := frontmatter.Parse(bytes.NewReader(content), &blog)
	if err != nil {
		return ctx.SendStatus(http.StatusInternalServerError) // err
	}

	if blog.Date == nil || blog.Date.After(time.Now()) {
		return ctx.SendStatus(http.StatusNotFound)
	}

	blog.Slug = slug

	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(rest), &buf); err != nil {
		return ctx.SendStatus(http.StatusInternalServerError)
	}
	blog.Content = buf.String()

	ctx.Locals("blog", &blog)
	return ctx.Next()
}
