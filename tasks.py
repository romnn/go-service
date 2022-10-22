from invoke import task
from pathlib import Path

PKG = "github.com/romnn/go-service"
CMD_PKG = PKG

ROOT_DIR = Path(__file__).parent
BUILD_DIR = ROOT_DIR / "build"


@task
def format(c):
    """Format code"""
    c.run("pre-commit run go-fmt --all-files")
    c.run("pre-commit run go-imports --all-files")


@task
def embed(c):
    """Embeds the examples"""
    c.run(f"npx embedme {ROOT_DIR/ 'README.md'}")


@task
def test(c):
    """Run tests"""
    c.run("env GO111MODULE=on go test -v -race ./...")


@task
def cyclo(c):
    """Check code complexity"""
    c.run("pre-commit run go-cyclo --all-files")


@task
def lint(c):
    """Lint code"""
    c.run("pre-commit run go-lint --all-files")
    c.run("pre-commit run go-vet --all-files")


@task
def install_hooks(c):
    """Install pre-commit hooks"""
    c.run("pre-commit install")


@task
def pre_commit(c):
    """Run all pre-commit checks"""
    c.run("pre-commit run --all-files")


@task
def coverage(c):
    """Create coverage report"""
    cmd = [
        "env",
        "GO111MODULE=on",
        "go",
        "test",
        "-race",
        "-coverprofile=coverage.txt",
        "-coverpkg=all",
        "-covermode=atomic",
        "./...",
    ]
    c.run(" ".join(cmd))


@task
def build(c):
    """Build the project"""
    c.run("pre-commit run go-build --all-files")


@task
def compile_proto(c):
    """Build the project"""
    from pprint import pprint

    services = [
        ROOT_DIR / "examples" / "grpc" / "service.proto",
        ROOT_DIR / "examples" / "auth" / "service.proto",
        ROOT_DIR / "examples" / "reflect" / "service.proto",
    ]
    for service in services:
        proto_path = service.parent
        out_dir = proto_path / "gen"
        out_dir.mkdir(parents=True, exist_ok=True)
        print(
            f"compiling {service.relative_to(ROOT_DIR)} to {out_dir.relative_to(ROOT_DIR)}"
        )
        package = (
            f"{service.relative_to(proto_path)}="
            + f"github.com/{out_dir.relative_to(ROOT_DIR.parent)}"
        )
        cmd = [
            "protoc",
            f"--proto_path={proto_path}",
            f"--go_opt=M{package}",
            f"--go-grpc_opt=M{package}",
            f"--go_out={out_dir}",
            f"--go-grpc_out={out_dir}",
            "--go_opt=paths=source_relative",
            "--go-grpc_opt=paths=source_relative",
            str(service),
        ]
        # pprint(cmd)
        c.run(" ".join(cmd))


@task
def clean_build(c):
    """Clean up files from package building"""
    c.run("rm -fr build/")


@task
def clean_coverage(c):
    """Clean up files from coverage measurement"""
    c.run("find . -name 'coverage.txt' -exec rm -fr {} +")


@task(pre=[clean_build, clean_coverage])
def clean(c):
    """Runs all clean sub-tasks"""
    pass
