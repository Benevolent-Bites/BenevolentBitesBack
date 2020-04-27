import click
import requests
import json

@click.command()
@click.option("--url", help="url of Google API call")
@click.option("--params", help="json struct of params to api call")
def main(url, params):
    j = json.loads(params)
    r = requests.get(url, params=j)
    click.echo(r.json())

if __name__ == "__main__":
    main()