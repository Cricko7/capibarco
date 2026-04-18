import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import '../../../app/localization/app_localizations.dart';
import '../../../shared/presentation/page_shell.dart';
import '../../../shared/presentation/section_header.dart';
import '../../../shared/presentation/soft_card.dart';
import '../../../shared/presentation/status_view.dart';
import '../../auth/presentation/auth_controller.dart';
import '../domain/entities/chat_message.dart';
import 'chat_repository_provider.dart';

class ChatPage extends ConsumerStatefulWidget {
  const ChatPage({
    required this.conversationId,
    required this.title,
    super.key,
  });

  final String conversationId;
  final String title;

  @override
  ConsumerState<ChatPage> createState() => _ChatPageState();
}

class _ChatPageState extends ConsumerState<ChatPage> {
  final _messageController = TextEditingController();
  List<ChatMessageEntity> _messages = const <ChatMessageEntity>[];
  bool _isLoading = true;
  bool _isSending = false;
  String? _errorMessage;

  @override
  void initState() {
    super.initState();
    Future<void>.microtask(_loadMessages);
  }

  @override
  void dispose() {
    _messageController.dispose();
    super.dispose();
  }

  Future<void> _loadMessages() async {
    setState(() {
      _isLoading = true;
      _errorMessage = null;
    });
    try {
      final messages = await ref
          .read(chatRepositoryProvider)
          .listMessages(widget.conversationId);
      if (!mounted) {
        return;
      }
      setState(() {
        _messages = messages;
        _isLoading = false;
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _errorMessage = error.toString();
        _isLoading = false;
      });
    }
  }

  Future<void> _sendMessage() async {
    final text = _messageController.text.trim();
    if (text.isEmpty || _isSending) {
      return;
    }

    setState(() => _isSending = true);
    try {
      final message = await ref
          .read(chatRepositoryProvider)
          .sendMessage(conversationId: widget.conversationId, text: text);
      if (!mounted) {
        return;
      }
      _messageController.clear();
      setState(() {
        _messages = <ChatMessageEntity>[..._messages, message];
        _isSending = false;
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() => _isSending = false);
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text(error.toString())));
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final currentProfileId =
        ref.watch(authControllerProvider).session?.user.id ?? '';
    final title = widget.title.isEmpty ? l10n.chat : widget.title;

    return Scaffold(
      body: PageShell(
        child: Column(
          children: <Widget>[
            SectionHeader(title: title, subtitle: l10n.chatReady),
            const SizedBox(height: 16),
            Expanded(
              child: _isLoading && _messages.isEmpty
                  ? StatusView.loading(message: l10n.loading)
                  : _errorMessage != null && _messages.isEmpty
                  ? StatusView.message(
                      message: _errorMessage!,
                      icon: Icons.error_outline_rounded,
                      action: FilledButton(
                        onPressed: _loadMessages,
                        child: Text(l10n.retry),
                      ),
                    )
                  : ListView.separated(
                      itemCount: _messages.length,
                      separatorBuilder: (_, _) => const SizedBox(height: 10),
                      itemBuilder: (context, index) {
                        final message = _messages[index];
                        final isMine =
                            message.senderProfileId == currentProfileId;
                        return Align(
                          alignment: isMine
                              ? Alignment.centerRight
                              : Alignment.centerLeft,
                          child: ConstrainedBox(
                            constraints: const BoxConstraints(maxWidth: 340),
                            child: DecoratedBox(
                              decoration: BoxDecoration(
                                color: isMine
                                    ? Theme.of(context).colorScheme.primary
                                    : Theme.of(
                                        context,
                                      ).colorScheme.surfaceContainerHighest,
                                borderRadius: BorderRadius.circular(24),
                              ),
                              child: Padding(
                                padding: const EdgeInsets.symmetric(
                                  horizontal: 16,
                                  vertical: 12,
                                ),
                                child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: <Widget>[
                                    Text(
                                      message.text,
                                      style: Theme.of(context)
                                          .textTheme
                                          .bodyLarge
                                          ?.copyWith(
                                            color: isMine
                                                ? Theme.of(
                                                    context,
                                                  ).colorScheme.onPrimary
                                                : null,
                                          ),
                                    ),
                                    if (message.sentAt != null) ...<Widget>[
                                      const SizedBox(height: 6),
                                      Text(
                                        DateFormat.Hm().format(
                                          message.sentAt!.toLocal(),
                                        ),
                                        style: Theme.of(context)
                                            .textTheme
                                            .labelSmall
                                            ?.copyWith(
                                              color: isMine
                                                  ? Theme.of(context)
                                                        .colorScheme
                                                        .onPrimary
                                                        .withValues(alpha: 0.82)
                                                  : Theme.of(context)
                                                        .colorScheme
                                                        .onSurfaceVariant,
                                            ),
                                      ),
                                    ],
                                  ],
                                ),
                              ),
                            ),
                          ),
                        );
                      },
                    ),
            ),
            const SizedBox(height: 16),
            SoftCard(
              child: Row(
                children: <Widget>[
                  Expanded(
                    child: TextField(
                      controller: _messageController,
                      minLines: 1,
                      maxLines: 4,
                      decoration: InputDecoration(
                        hintText: l10n.messageHint,
                        border: InputBorder.none,
                      ),
                    ),
                  ),
                  const SizedBox(width: 12),
                  FilledButton.icon(
                    onPressed: _isSending ? null : _sendMessage,
                    icon: _isSending
                        ? const SizedBox(
                            width: 16,
                            height: 16,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : const Icon(Icons.send_rounded),
                    label: Text(l10n.send),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
