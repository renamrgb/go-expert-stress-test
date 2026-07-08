# go-expert-stress-test

CLI em Go para testes de carga (stress test) contra serviços web via HTTP. A ferramenta
dispara um número configurável de requisições `GET` concorrentes contra uma URL alvo e, ao
final da execução, imprime um relatório com o tempo total, o total de requisições realizadas
e a distribuição de códigos de status HTTP recebidos.

## Funcionalidades

- Número total de requisições e nível de concorrência configuráveis via flags.
- Distribuição de requisições entre workers concorrentes usando goroutines e channels.
- Garantia de que o número de requisições executadas é **exatamente** o solicitado.
- Relatório final com:
  - Tempo total de execução;
  - Total de requisições realizadas;
  - Quantidade de respostas HTTP 200;
  - Distribuição de todos os demais códigos de status (404, 500, etc.), agrupados por código;
  - Falhas de rede/conexão (quando não há resposta HTTP), contadas separadamente.
- CLI construída com [Cobra](https://github.com/spf13/cobra), a ferramenta padrão do
  ecossistema Go para aplicações de linha de comando.

## Estrutura do projeto

```
.
├── main.go                       # ponto de entrada: apenas chama cmd.Execute()
├── cmd/
│   └── root.go                   # definição do comando Cobra e suas flags
├── internal/
│   ├── httpclient/               # execução das requisições HTTP GET
│   │   └── httpclient.go
│   ├── loadtest/                 # lógica de concorrência do teste de carga
│   │   └── loadtest.go
│   └── report/                   # agregação e impressão do relatório final
│       └── report.go
├── Dockerfile                    # build multi-stage, imagem final "scratch"
├── go.mod
├── go.sum
└── README.md
```

## Pré-requisitos

- [Go](https://go.dev/dl/) 1.23 ou superior (apenas para build/execução local sem Docker).
- [Docker](https://docs.docker.com/get-docker/) (para build e execução via container).

A única dependência externa é [`github.com/spf13/cobra`](https://github.com/spf13/cobra)
(mais suas dependências transitivas `spf13/pflag` e `inconshreveable/mousetrap`), usada para
o parsing de flags e a estrutura de comando da CLI. Toda a lógica de teste de carga usa
apenas a biblioteca padrão do Go.

## Parâmetros

| Flag             | Obrigatório | Descrição                                             |
|------------------|:-----------:|--------------------------------------------------------|
| `--url`          | Sim         | URL do serviço a ser testado (ex: `http://google.com`) |
| `--requests`     | Sim         | Número total de requisições a serem realizadas         |
| `--concurrency`  | Não (padrão `1`) | Número de chamadas simultâneas                    |

## Executando localmente (via CLI)

```bash
go run . --url=http://google.com --requests=1000 --concurrency=10
```

Ou, compilando o binário primeiro:

```bash
go build -o stress-test .
./stress-test --url=http://google.com --requests=1000 --concurrency=10
```

### Exemplo de saída

```
Starting load test: url=http://google.com requests=1000 concurrency=10

==================================================
               STRESS TEST REPORT
==================================================
Total execution time : 1.234s (1.234s)
Total requests made  : 1000
Requests with status 200 OK : 987
--------------------------------------------------
Distribution of other HTTP status codes:
  HTTP 301: 10
  HTTP 404: 3
==================================================
```

## Build e execução via Docker

### 1. Buildar a imagem

A partir da raiz do repositório:

```bash
docker build -t go-expert-stress-test .
```

Isso executa um build multi-stage: a primeira etapa compila o binário Go estático e a
segunda copia apenas o binário e os certificados TLS para uma imagem final `scratch`
(minimalista, sem shell nem pacotes extras).

### 2. Executar o container

```bash
docker run go-expert-stress-test --url=http://google.com --requests=1000 --concurrency=10
```

O `ENTRYPOINT` do container já aponta para o binário, então todos os argumentos passados
após o nome da imagem são repassados diretamente como flags da aplicação.

> Ao testar contra um serviço rodando na própria máquina host (ex: `http://localhost:8080`),
> adicione `--network host` (Linux) ou aponte para `http://host.docker.internal:8080`
> (Docker Desktop no macOS/Windows), já que `localhost` dentro do container se refere ao
> próprio container.

## Validações

A aplicação valida os parâmetros de entrada antes de iniciar o teste:

- `--url` e `--requests` são obrigatórios (o Cobra rejeita a execução se não forem informados).
- `--url` não pode ser vazio.
- `--requests` deve ser um inteiro maior que zero.
- `--concurrency` deve ser um inteiro maior que zero.

Em caso de parâmetro inválido, a aplicação imprime a mensagem de erro e encerra com código
de saída `1`. Use `--help` para ver o modo de uso completo.

## Detalhes de implementação

- As requisições são distribuídas usando um channel pré-carregado com um "token" por
  requisição a ser executada; um pool fixo de goroutines (`--concurrency`) consome esse
  channel até ele se esgotar, garantindo que o total de requisições executadas seja
  exatamente o valor de `--requests`, independentemente do nível de concorrência.
- O cliente HTTP segue redirects automaticamente (comportamento padrão do `net/http`, até
  10 saltos), o mesmo que `curl -L` ou um navegador fariam. É por isso que testar
  `http://google.com` mostra respostas `200`: a URL literal sempre responde `301` para
  `https://www.google.com/`, e é a página final que retorna `200`.
- Falhas de rede genuínas (timeout, conexão recusada, DNS, excesso de redirects, etc.) não
  geram um código de status HTTP e são contabilizadas separadamente no relatório como
  "Network/connection errors". Ao testar serviços reais de produção (como `google.com`) sob
  concorrência alta, é esperado ver uma pequena porcentagem desses erros: provedores como o
  Google aplicam proteção anti-bot/anti-abuso e ocasionalmente redirecionam rajadas de
  tráfego automatizado para uma página de verificação, gerando um loop de redirecionamento
  que estoura o limite de 10 saltos. Isso não é um bug da ferramenta — é o comportamento real
  do alvo sob carga, que é exatamente o tipo de informação que um teste de stress deve
  reportar. Para uma demonstração sem esse ruído, use um serviço próprio ou um endpoint de
  teste estável (ex: `https://httpbin.org/get`).

> **Nota sobre testar `http://google.com` via Docker:** em ambientes de rede compartilhada
> (containers, VMs de CI, sandboxes de nuvem) é comum o IPv4 de saída já estar
> marcado/limitado por proteções anti-abuso do Google, enquanto a máquina host, se tiver
> conectividade IPv6 própria, escapa desse limite. Se `docker run ... --url=http://google.com`
> retornar consistentemente `HTTP 429` para todas as requisições, isso normalmente indica que
> a rede Docker do ambiente não tem saída IPv6 e está usando um IPv4 compartilhado já
> throttled — não é um defeito da aplicação (o status `429` real está sendo corretamente
> capturado e reportado). Para validar a ferramenta sem esse ruído de rede, use um alvo
> estável como `https://httpbin.org/get` ou um servidor HTTP próprio.
- Timeout por requisição de 30s para evitar que uma requisição travada impeça o teste de
  terminar.
