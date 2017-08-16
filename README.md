### About the Project

A Google Drive to static site generator via [Hugo](https://gohugo.io/), [Pandoc](https://github.com/jgm/pandoc), and [Drive](https://github.com/odeke-em/drive). Created for [Google Summer of Code 2017](https://developers.google.com/open-source/gsoc/) under the supervision of [Portland State University](http://wiki.cs.pdx.edu/psu-gsoc/index.html). Here is the [Google Summer of Code Project Site](https://summerofcode.withgoogle.com/projects/#5859356254928896)

#### About the name

driveraker is styled without capitalization. The name derives from the term [muckraker](https://en.wikipedia.org/wiki/Muckraker), an insult (or compliment) directed at journalists, I named driveraker such because driveraker aims to integrate Google Drive with a static site generator like Hugo in order to provide a website for a news publication.

You can read the changelog [here.](https://gatlinnewhouse.github.io/driveraker/changelog) And downloads can be found [here.](https://github.com/gatlinnewhouse/driveraker/releases/tag/v0.1-alpha)

### Installation

Requirements:

* A linux distro of choice with systemd as the init system.
* [Pandoc](https://github.com/jgm/pandoc) with version >= 1.19.2.1
* [Go](https://golang.org/doc/install) with version >= 1.8
* [drive](https://github.com/odeke-em/drive)
* [Hugo](https://github.com/spf13/hugo)
* A http(s) server like [nginx](https://nginx.org/) or [apache](https://httpd.apache.org/)

#### Installation instructions

WIP

### How to use driveraker

### Google Document Template

driveraker relies on Google Documents synced to have a specific format for certain aspect of the article processing. [Here](https://docs.google.com/document/d/1HQcVevrXg_uwtdJy5ulfl9vN31yEieZeNyN7yqJQe8w/edit?usp=sharing) is an example of a template document for driveraker.

As you can see: tags, categories, publication date, and updated date values are listed at the top of the file with corresponding variables.

* `DRVRKR_TAGS:` is the variable written before tags are listed. This will appear as tags on the article which can be clicked to find other articles with the same tags
* `DRVRKR_CATEGORIES:` is the variable written before categories are listed. This will appear as the categories of an article which can be clicked to find other articles with the same categories
* `DRVRKR_PUB_DATE:` is the variable written before the publication date of the article. This will appear as the publication date “Published on May 4, 2017”
* `DRVRKR_UPDATE_DATE:` is the variable written before the date an article was updated. This will appear as the updated date “Updated on June 1, 2017”

driveraker also relies on the Google Document synced to convert the header (title style in Google Drive) into the first Markdown header so it can be made into the article title on the webpage. This logic extends to the subheader (or subtitle style in Google Drive), image caption (or heading 3 style in Google Drive), byline (or heading 1 style in Google Drive), and in-article header (or heading 2 style in Google Drive).

In order to preserve image quality it’s recommended that images are uploaded to Google Drive and then inserted into the article either under the driveraker variables on a new line for an image appearing at the beginning of an article or as a thumbnail, or within the article for an inline image.

The byline of an article is also special since driveraker will parse all the names after “By” with respect to commas/and in order to prepend the names of the authors into the article’s webpage to link to their author pages.
