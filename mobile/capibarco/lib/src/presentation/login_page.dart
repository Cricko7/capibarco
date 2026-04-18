import 'package:flutter/material.dart';

import 'login_controller.dart';

class LoginPage extends StatefulWidget {
  const LoginPage({required this.controller, super.key});

  final LoginController controller;

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  final _tenantController = TextEditingController(text: 'default');
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();

  @override
  void initState() {
    super.initState();
    widget.controller.checkReadiness();
    widget.controller.addListener(_onChanged);
  }

  @override
  void dispose() {
    widget.controller.removeListener(_onChanged);
    _tenantController.dispose();
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  void _onChanged() {
    setState(() {});
  }

  @override
  Widget build(BuildContext context) {
    final session = widget.controller.session;

    return Scaffold(
      appBar: AppBar(title: const Text('Capibarco Auth')),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              widget.controller.backendReady
                  ? 'Backend ready'
                  : 'Backend not ready',
              style: TextStyle(
                color: widget.controller.backendReady
                    ? Colors.green
                    : Colors.orange,
              ),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _tenantController,
              decoration: const InputDecoration(labelText: 'Tenant ID'),
            ),
            TextField(
              controller: _emailController,
              decoration: const InputDecoration(labelText: 'Email'),
            ),
            TextField(
              controller: _passwordController,
              decoration: const InputDecoration(labelText: 'Password'),
              obscureText: true,
            ),
            const SizedBox(height: 16),
            ElevatedButton(
              onPressed: widget.controller.isLoading
                  ? null
                  : () {
                      widget.controller.login(
                        tenantId: _tenantController.text,
                        email: _emailController.text,
                        password: _passwordController.text,
                      );
                    },
              child: widget.controller.isLoading
                  ? const CircularProgressIndicator()
                  : const Text('Login'),
            ),
            if (widget.controller.error != null)
              Padding(
                padding: const EdgeInsets.only(top: 12),
                child: Text(
                  widget.controller.error!,
                  style: const TextStyle(color: Colors.red),
                ),
              ),
            if (session != null)
              Padding(
                padding: const EdgeInsets.only(top: 12),
                child: Text('Logged in as ${session.email} (${session.userId})'),
              ),
          ],
        ),
      ),
    );
  }
}
