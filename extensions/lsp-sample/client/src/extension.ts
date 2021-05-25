/* --------------------------------------------------------------------------------------------
 * Copyright (c) Microsoft Corporation. All rights reserved.
 * Licensed under the MIT License. See License.txt in the project root for license information.
 * ------------------------------------------------------------------------------------------ */

import * as path from 'path';
import { workspace, ExtensionContext } from 'vscode';

import {
	LanguageClient,
	LanguageClientOptions,
	ServerOptions,
	TransportKind
} from 'vscode-languageclient/node';

import {
	CancellationToken,
	CloseAction,
	CompletionItemKind,
	ConfigurationParams,
	ConfigurationRequest,
	ErrorAction,
	ExecuteCommandSignature,
	HandleDiagnosticsSignature,
	InitializeError,
	Message,
	ProvideCodeLensesSignature,
	ProvideCompletionItemsSignature,
	ProvideDocumentFormattingEditsSignature,
	ResponseError,
	RevealOutputChannelOn
} from 'vscode-languageclient';

import vscode = require('vscode');

let client: LanguageClient;
let crashCount = 0;
export let serverOutputChannel: vscode.OutputChannel;

export function activate(context: ExtensionContext) {
	// The server is implemented in node
	const serverModule = context.asAbsolutePath(
		path.join('server', 'qlsp')
	);
	// The debug options for the server
	// --inspect=6009: runs the server in Node's Inspector mode so VS Code can attach to the server for debugging
	const debugOptions = { execArgv: ['--nolazy', '--inspect=6009'] };

	// If the extension is launched in debug mode then the debug server options are used
	// Otherwise the run options are used
	const serverOptions: ServerOptions = {
		run: { module: serverModule, transport: TransportKind.socket },
		debug: {
			module: serverModule,
			transport: TransportKind.socket,
			options: debugOptions
		}
	};

	// Options to control the language client
	const clientOptions: LanguageClientOptions = {
		// Register the server for plain text documents
		documentSelector: [{ scheme: 'file', language: 'sql' }],
		synchronize: {
			// Notify the server about file changes to '.clientrc files contained in the workspace
			fileEvents: workspace.createFileSystemWatcher('**/.sql')
		},
		outputChannel: vscode.window.createOutputChannel('qlsp (server)'),
		traceOutputChannel: vscode.window.createOutputChannel('qlsp'),
		errorHandler: {
			error: (error: Error, message: Message, count: number): ErrorAction => {
				// Allow 5 crashes before shutdown.
				if (count < 15) {
					return ErrorAction.Continue;
				}
				vscode.window.showErrorMessage(
					`Error communicating with the language server: ${error}: ${message}.`
				);
				return ErrorAction.Shutdown;
			},
			closed: (): CloseAction => {
				// Allow 5 crashes before shutdown.
				crashCount++;
				if (crashCount < 15) {
					return CloseAction.Restart;
				}

				return CloseAction.DoNotRestart;
			}
		},
		middleware: {
			executeCommand: async (command: string, args: any[], next: ExecuteCommandSignature) => {
				try {
					return await next(command, args);
				} catch (e) {
					const answer = await vscode.window.showErrorMessage(
						`Command '${command}' failed: ${e}.`,
						'Show Trace'
					);
					if (answer === 'Show Trace') {
						serverOutputChannel.show();
					}
					return null;
				}
			}
		}
	};

	// Create the language client and start the client.
	client = new LanguageClient(
		'languageServerExample',
		'Language Server Example',
		{
			command: path.resolve(__dirname, '..', '..', 'server', 'genjilsp')
		},
		clientOptions
	);

	// Start the client. This will also launch the server
	client.start();
}

export function deactivate(): Thenable<void> | undefined {
	if (!client) {
		return undefined;
	}
	return client.stop();
}
