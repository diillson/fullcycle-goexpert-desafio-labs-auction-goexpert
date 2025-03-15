## Documentação de Execução do Projeto
    
### Requisitos
    
- Docker e Docker Compose instalados
- Git
    
### Como executar o projeto em ambiente de desenvolvimento
    
1. Clone o repositório:
  ```bash
  git clone https://github.com/diillson/fullcycle-goexpert-desafio-labs-auction-goexpert.git
  cd fullcycle-goexpert-desafio-labs-auction-goexpert
```
2. Inicie os contêineres:
```bash
docker-compose up -d
```

3. Verifique os logs para garantir que a aplicação está funcionando:
```bash
docker-compose logs -f app
```

4. A API estará disponível em http://localhost:8080

### Endpoints disponíveis:
```text
•  GET /auction  - Listar leilões
•  GET /auction/:auctionId  - Buscar leilão por ID
•  POST /auction  - Criar novo leilão
•  GET /auction/winner/:auctionId  - Buscar lance vencedor de um leilão
•  POST /bid  - Criar novo lance
•  GET /bid/:auctionId  - Buscar lances de um leilão
•  GET /user/:userId  - Buscar usuário por ID
•  POST /user  - Criar novo usuário
```

### Criação de Usuários

Para facilitar os testes, o sistema agora inclui um endpoint para criar usuários:
```bash
    curl -X POST http://localhost:8080/user \
      -H "Content-Type: application/json" \
      -d '{
        "name": "Nome do Usuário"
      }'
```

Resposta:
```json
    {
      "id": "12345678-1234-1234-1234-123456789012",
      "name": "Nome do Usuário"
    }
```

O ID retornado pode ser usado para fazer lances nos leilões.

### Exemplo de criação de leilão:
```bash
    curl -X POST http://localhost:8080/auction \
      -H "Content-Type: application/json" \
      -d '{
        "product_name": "Smartphone XYZ",
        "category": "Electronics",
        "description": "Brand new smartphone with great features",
        "condition": 1
      }'
```

### Executando testes com Docker
    
Para facilitar a execução dos testes, forneço uma configuração Docker específica para testes. 
Você pode executar todos os testes com um único comando basta dar permissào de execução chmod +x ./run-test.sh:
    
```bash
    ./run-tests.sh
```

O teste demonstra a principal funcionalidade implementada: o fechamento automático de leilões
após o intervalo de tempo configurado.

### Explicação do comportamento de fechamento automático:

1. Quando um leilão é criado, ele é registrado em um mapa de leilões ativos com seu tempo de expiração
2. Uma goroutine em segundo plano verifica periodicamente (a cada 10 segundos) os leilões ativos
3. Se um leilão atingiu seu tempo limite (baseado na variável de ambiente  AUCTION_INTERVAL ), ele é automaticamente fechado
4. O status do leilão é atualizado no banco de dados para  Completed
5. Após o fechamento, novos lances não serão mais aceitos para esse leilão

## Exemplos de Lances

Aqui estão alguns exemplos de comandos  curl  para criar novos lances usando o endpoint  POST /bid :

## Exemplo 1: Lance básico
```bash
    curl -X POST http://localhost:8080/bid \
      -H "Content-Type: application/json" \
      -d '{
        "user_id": "11111111-1111-1111-1111-111111111111",
        "auction_id": "22222222-2222-2222-2222-222222222222",
        "amount": 100.50
      }'
```

Este exemplo envia um lance básico com:
```text
•  user_id : ID do usuário que está fazendo o lance
•  auction_id : ID do leilão onde o lance está sendo feito
•  amount : Valor do lance (100.50)
```

## Exemplo 2: Lance com valor mais alto
```bash
    curl -X POST http://localhost:8080/bid \
      -H "Content-Type: application/json" \
      -d '{
        "user_id": "33333333-3333-3333-3333-333333333333",
        "auction_id": "22222222-2222-2222-2222-222222222222",
        "amount": 150.75
      }'
```

## Exemplo 3: Lance com outro usuário
```bash
    curl -X POST http://localhost:8080/bid \
      -H "Content-Type: application/json" \
      -d '{
        "user_id": "44444444-4444-4444-4444-444444444444",
        "auction_id": "22222222-2222-2222-2222-222222222222",
        "amount": 200.00
      }'
```

## Exemplo 4: Lance para outro leilão
```bash
    curl -X POST http://localhost:8080/bid \
      -H "Content-Type: application/json" \
      -d '{
        "user_id": "11111111-1111-1111-1111-111111111111",
        "auction_id": "55555555-5555-5555-5555-555555555555",
        "amount": 75.25
      }'
```

## Observações importantes:

1. Os valores de  user_id  e  auction_id  precisam ser UUIDs válidos conforme a validação no código
2. Para testar corretamente, você precisa:
```text
   • Primeiro criar um leilão usando o endpoint  POST /auction
   • Usar o ID retornado desse leilão para o  auction_id  no lance
   • Criar um usuário válido ou usar um existente para o  user_id
```

3. O  amount  deve ser um valor positivo conforme validação no código

## Fluxo completo para testar:

### Informações sobre status e condição dos leilões:

1.  status : Indica se o leilão está aberto ou fechado
```text
    • Valor  0 : Leilão ativo/aberto (aceitando lances)
    • Valor  1 : Leilão completado/fechado (não aceita mais lances)
```
2.  condition : Refere-se à condição do produto sendo leiloado
```text
    • Valor  1 : Produto novo
    • Valor  2 : Produto usado
    • Valor  3 : Produto recondicionado/refurbished
```

### Passos para testar o sistema completo:

1. Criar um usuário:
```bash
    curl -X POST http://localhost:8080/user \
      -H "Content-Type: application/json" \
      -d '{
        "name": "Usuário de Teste"
      }'
```

Guarde o  id  retornado para usar nos lances.

2. Criar um leilão:
```bash
    curl -X POST http://localhost:8080/auction \
      -H "Content-Type: application/json" \
      -d '{
        "product_name": "Smartphone XYZ",
        "category": "Electronics",
        "description": "Brand new smartphone with great features",
        "condition": 1
      }'
```


3. Listar leilões para obter o ID:
```bash
    curl -X GET http://localhost:8080/auction?status=0
```

4. Fazer um lance usando o ID do usuário e do leilão:
```bash
    curl -X POST http://localhost:8080/bid \
      -H "Content-Type: application/json" \
      -d '{
        "user_id": "ID_DO_USUARIO_OBTIDO_NO_PASSO_1",
        "auction_id": "ID_DO_LEILAO_OBTIDO_NO_PASSO_3",
        "amount": 100.50
      }'
```

5. Verificar os lances do leilão:
```bash
    curl -X GET http://localhost:8080/bid/ID_DO_LEILAO_OBTIDO_NO_PASSO_3
```

6. Verificar o lance vencedor (após o leilão ser fechado automaticamente):
```bash
    curl -X GET http://localhost:8080/auction/winner/ID_DO_LEILAO_OBTIDO_NO_PASSO_3
```
