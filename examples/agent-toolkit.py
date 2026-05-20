"""Drop-in Agno Toolkit wrapping agno-docs-pp-cli.

Copy this file into your Agno project, instantiate the toolkit, and pass it
to any Agent or Team. Subprocess-isolated, async-safe.
"""

import asyncio
import json
import subprocess

from agno.tools import Toolkit

CLI = "agno-docs-pp-cli"


def _run(*args: str) -> str:
    """Run the CLI with --json and return stdout (or JSON error)."""
    try:
        r = subprocess.run(
            [CLI, *args, "--json"],
            capture_output=True,
            text=True,
            timeout=15,
        )
        if r.returncode != 0:
            return json.dumps({"error": r.stderr.strip() or "CLI error"})
        return r.stdout.strip()
    except FileNotFoundError:
        return json.dumps(
            {"error": f"{CLI} not found. Install: "
                      f"go install -tags sqlite_fts5 "
                      f"github.com/sekai1710/agno-docs-pp-cli/cmd/agno-docs-pp-cli@latest"}
        )
    except subprocess.TimeoutExpired:
        return json.dumps({"error": "CLI timeout after 15s"})


class AgnoDocsTool(Toolkit):
    """Grounded lookup over docs.agno.com — start with `agno_docs_which`."""

    def __init__(self) -> None:
        super().__init__(name="agno_docs")
        self.register(self.agno_docs_which)
        self.register(self.agno_docs_context)
        self.register(self.agno_docs_examples)
        self.register(self.agno_docs_sections)

    async def agno_docs_which(self, query: str, limit: int = 5) -> str:
        """Find Agno docs pages covering a topic. Call BEFORE answering any
        Agno question (Agent, Team, Workflow, AgentOS, Knowledge, Tools,
        Models providers, DB backends, memory, embedders).

        Args:
            query: Natural language query, e.g. 'how do teams work',
                   'PostgresDb config', 'OpenRouter model'
            limit: Max matches (default 5)
        """
        return await asyncio.to_thread(_run, "which", query, "--limit", str(limit))

    async def agno_docs_context(self, slug: str) -> str:
        """Get the full markdown body of an Agno docs page.

        Args:
            slug: Page slug (e.g. 'agents', 'teams') OR full URL —
                  both work; URLs are normalized to slug automatically.
        """
        if slug.startswith("http"):
            slug = slug.rstrip("/").split("/")[-1]
        return await asyncio.to_thread(_run, "context", slug)

    async def agno_docs_examples(
        self, query: str, language: str = "python", limit: int = 3
    ) -> str:
        """Extract paste-ready code examples from Agno docs.

        Args:
            query: e.g. 'team coordinate', 'agent with PostgresDb'
            language: 'python' (default), 'bash', 'json', 'yaml', or ''
            limit: Max examples (default 3)
        """
        args = ["examples", query, "--limit", str(limit)]
        if language:
            args += ["--language", language]
        return await asyncio.to_thread(_run, *args)

    async def agno_docs_sections(self) -> str:
        """List all Agno docs sections with page counts."""
        return await asyncio.to_thread(_run, "sections")


# Example usage:
if __name__ == "__main__":
    from agno.agent import Agent
    from agno.models.openrouter import OpenRouter

    async def main() -> None:
        agent = Agent(
            model=OpenRouter(id="google/gemini-2.5-flash"),
            tools=[AgnoDocsTool()],
            instructions=[
                "You answer questions about the Agno framework.",
                "ALWAYS call agno_docs_which first to find the right page.",
                "Cite the source URL in your answer.",
            ],
            markdown=True,
        )
        response = await agent.arun(
            "How do I create a Team in Agno with coordinate mode? Show python code."
        )
        print(response.content)

    asyncio.run(main())
