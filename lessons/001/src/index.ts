import fastify, { FastifyInstance } from 'fastify';
import axios from 'axios';

const server: FastifyInstance = fastify({ logger: true });

interface Todo {
  id: number;
  title: string;
  completed: boolean;
}

server.get('/', async (request, reply) => {
  const url = 'https://jsonplaceholder.typicode.com/todos/1';
  const response = await axios.get<Todo>(url);
  const todo = response.data;

  const loggedTodo = logTodo(todo.id, todo.title, todo.completed);
  return { message: 'Todo fetched successfully', todo: loggedTodo };
});

const logTodo = (id: number, title: string, completed: boolean): Todo => {
  const loggedTodo: Todo = { id, title, completed };
  console.log(`
    The Todo with ID: ${id}
    Has a title of: ${title}
    Is it finished? ${completed}
  `);
  return loggedTodo;
};

const start = async () => {
  try {
    await server.listen({ port: 3000 });
  } catch (err) {
    server.log.error(err);
    process.exit(1);
  }
};

start();
