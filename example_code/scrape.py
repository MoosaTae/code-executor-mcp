import requests
from bs4 import BeautifulSoup
# Send a request to Hacker News
response = requests.get("https://news.ycombinator.com/")
soup = BeautifulSoup(response.text, "html.parser")
# Find all story titles and links
items = []
titles = soup.select("span.titleline > a")
scores = soup.select("span.score")
score_index = 0
for i, title in enumerate(titles[:10]):
  # Get the score if available
  score_text = "No score"
  if score_index < len(scores):
    subtext = scores[score_index].text
    score_text = subtext
    score_index += 1

    # Get the link
    link = title["href"]
    if not link.startswith("http"):
      link = f"https://news.ycombinator.com/{link}"

    items.append({
      "title": title.text,
      "link": link,
      "score": score_text
    })

    # Print the top 10 articles
    print("# Top 10 Hacker News Articles\
      ")
    for i, item in enumerate(items, 1):
      print(f"{i}. **{item['title']}**")
      print(f"   - Score: {item['score']}")
      print(f"   - Link: {item['link']}")
      print()
