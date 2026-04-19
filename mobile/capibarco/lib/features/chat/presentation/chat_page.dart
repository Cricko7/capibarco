import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import '../../../app/localization/app_localizations.dart';
import '../../../bootstrap/providers.dart';
import '../../../core/config/environment.dart';
import '../../../core/network/network_providers.dart';
import '../../../core/network/rest_service_client.dart';
import '../../../shared/presentation/page_shell.dart';
import '../../../shared/presentation/section_header.dart';
import '../../../shared/presentation/soft_card.dart';
import '../../../shared/presentation/status_view.dart';
import '../../auth/presentation/auth_controller.dart';
import '../../profile/data/api/profile_api_client.dart';
import '../domain/entities/chat_message.dart';
import 'chat_repository_provider.dart';

class ChatPage extends ConsumerStatefulWidget {
  const ChatPage({
    required this.conversationId,
    required this.title,
    required this.returnTo,
    required this.counterpartProfileId,
    super.key,
  });

  final String conversationId;
  final String title;
  final String returnTo;
  final String counterpartProfileId;

  @override
  ConsumerState<ChatPage> createState() => _ChatPageState();
}

class _ChatPageState extends ConsumerState<ChatPage> {
  static final RegExp _uuidPattern = RegExp(
    r'^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$',
  );
  static final RegExp _chatWithUuidPattern = RegExp(
    r'^(?:Chat with|Чат с)\s+([0-9a-fA-F-]{36})$',
  );

  final _messageController = TextEditingController();
  final _messagesScrollController = ScrollController();
  StreamSubscription<ChatMessageEntity>? _messagesSubscription;
  List<ChatMessageEntity> _messages = const <ChatMessageEntity>[];
  bool _isLoading = true;
  bool _isSending = false;
  String? _errorMessage;
  String _resolvedCounterpartProfileId = '';
  String _resolvedTitle = '';
  String _counterpartAvatarUrl = '';

  @override
  void initState() {
    super.initState();
    _resolvedCounterpartProfileId = widget.counterpartProfileId.trim();
    _resolvedTitle = _sanitizeTitle(
      widget.title,
      counterpartProfileId: _resolvedCounterpartProfileId,
    );
    Future<void>.microtask(_initializeChat);
  }

  @override
  void dispose() {
    _messagesSubscription?.cancel();
    _messageController.dispose();
    _messagesScrollController.dispose();
    super.dispose();
  }

  Future<void> _initializeChat() async {
    unawaited(_loadCounterpartProfile());
    _subscribeToRealtime();
    await _loadMessages();
  }

  Future<void> _loadCounterpartProfile() async {
    var counterpartProfileId = _resolvedCounterpartProfileId;
    final currentProfileId =
        ref.read(authControllerProvider).session?.user.id ?? '';
    if (counterpartProfileId.isEmpty && currentProfileId.isNotEmpty) {
      try {
        final conversations = await ref
            .read(chatRepositoryProvider)
            .listConversations();
        for (final conversation in conversations) {
          if (conversation.id != widget.conversationId) {
            continue;
          }
          counterpartProfileId = conversation.counterpartProfileId(
            currentProfileId,
          );
          break;
        }
      } catch (_) {
        // The chat can still work without profile navigation metadata.
      }
    }

    if (counterpartProfileId.isEmpty) {
      if (!mounted || _resolvedCounterpartProfileId.isNotEmpty) {
        return;
      }
      setState(() => _resolvedCounterpartProfileId = '');
      return;
    }

    var nextTitle = _resolvedTitle;
    var nextAvatarUrl = _counterpartAvatarUrl;

    try {
      final environment = ref.read(appEnvironmentProvider);
      final profile = await ProfileApiClient(
        RestServiceClient(
          dio: ref.read(authenticatedDioProvider),
          config: environment.service(ServiceKind.profiles),
        ),
      ).getProfile(counterpartProfileId);
      final displayName = profile.displayName.trim();
      if (displayName.isNotEmpty) {
        nextTitle = displayName;
      }
      nextAvatarUrl = profile.avatarUrl.trim();
    } catch (_) {
      // Fallback to the safe title and icon-only avatar.
    }

    if (!mounted) {
      return;
    }
    setState(() {
      _resolvedCounterpartProfileId = counterpartProfileId;
      _resolvedTitle = nextTitle;
      _counterpartAvatarUrl = nextAvatarUrl;
    });
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
        _messages = _mergeMessages(_messages, messages);
        _isLoading = false;
      });
      _scrollToLatest(jumpToEnd: true);
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
        _messages = _mergeMessage(_messages, message);
        _isSending = false;
      });
      _scrollToLatest();
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

  void _subscribeToRealtime() {
    _messagesSubscription?.cancel();
    _messagesSubscription = ref
        .read(chatRepositoryProvider)
        .watchMessages(
          conversationId: widget.conversationId,
          accessToken:
              ref.read(authControllerProvider).session?.accessToken ?? '',
        )
        .listen((message) {
          if (!mounted) {
            return;
          }
          setState(() {
            _messages = _mergeMessage(_messages, message);
          });
          _scrollToLatest();
        });
  }

  List<ChatMessageEntity> _mergeMessages(
    List<ChatMessageEntity> existing,
    List<ChatMessageEntity> incoming,
  ) {
    var nextMessages = existing;
    for (final message in incoming) {
      nextMessages = _mergeMessage(nextMessages, message);
    }
    return nextMessages;
  }

  List<ChatMessageEntity> _mergeMessage(
    List<ChatMessageEntity> current,
    ChatMessageEntity incoming,
  ) {
    final updated = <ChatMessageEntity>[...current];
    final existingIndex = updated.indexWhere((item) => item.id == incoming.id);
    if (existingIndex >= 0) {
      updated[existingIndex] = incoming;
    } else {
      updated.add(incoming);
    }
    updated.sort(_compareMessages);
    return updated;
  }

  int _compareMessages(ChatMessageEntity left, ChatMessageEntity right) {
    final sentAtComparison =
        (left.sentAt ?? DateTime.fromMillisecondsSinceEpoch(0, isUtc: true))
            .compareTo(
              right.sentAt ??
                  DateTime.fromMillisecondsSinceEpoch(0, isUtc: true),
            );
    if (sentAtComparison != 0) {
      return sentAtComparison;
    }
    return left.id.compareTo(right.id);
  }

  void _scrollToLatest({bool jumpToEnd = false}) {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || !_messagesScrollController.hasClients) {
        return;
      }

      final maxScrollExtent =
          _messagesScrollController.position.maxScrollExtent;
      if (jumpToEnd) {
        _messagesScrollController.jumpTo(maxScrollExtent);
        return;
      }

      _messagesScrollController.animateTo(
        maxScrollExtent,
        duration: const Duration(milliseconds: 220),
        curve: Curves.easeOut,
      );
    });
  }

  Widget _buildComposer(AppLocalizations l10n) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final isCompact = constraints.maxWidth < 560;
        final composerField = TextField(
          controller: _messageController,
          minLines: 1,
          maxLines: 4,
          textInputAction: TextInputAction.send,
          onSubmitted: (_) => _sendMessage(),
          decoration: InputDecoration(
            hintText: l10n.messageHint,
            contentPadding: const EdgeInsets.symmetric(
              horizontal: 18,
              vertical: 16,
            ),
          ),
        );
        final composerButton = FilledButton.icon(
          onPressed: _isSending ? null : _sendMessage,
          icon: _isSending
              ? const SizedBox(
                  width: 16,
                  height: 16,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : const Icon(Icons.send_rounded),
          label: Text(l10n.send),
        );

        return SoftCard(
          padding: const EdgeInsets.all(12),
          child: isCompact
              ? Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: <Widget>[
                    composerField,
                    const SizedBox(height: 12),
                    composerButton,
                  ],
                )
              : Row(
                  crossAxisAlignment: CrossAxisAlignment.end,
                  children: <Widget>[
                    Expanded(child: composerField),
                    const SizedBox(width: 12),
                    composerButton,
                  ],
                ),
        );
      },
    );
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final currentProfileId =
        ref.watch(authControllerProvider).session?.user.id ?? '';
    final title = _resolvedTitle.isEmpty ? l10n.chat : _resolvedTitle;

    return Scaffold(
      body: PageShell(
        child: Column(
          children: <Widget>[
            Row(
              children: <Widget>[
                IconButton.filledTonal(
                  onPressed: () {
                    if (widget.returnTo.isNotEmpty) {
                      context.go(widget.returnTo);
                      return;
                    }
                    if (context.canPop()) {
                      context.pop();
                      return;
                    }
                    context.go('/discover');
                  },
                  icon: const Icon(Icons.arrow_back_rounded),
                  tooltip: 'Назад',
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: SectionHeader(title: title, subtitle: l10n.chatReady),
                ),
                if (_resolvedCounterpartProfileId.isNotEmpty) ...<Widget>[
                  const SizedBox(width: 12),
                  Tooltip(
                    message: l10n.openProfile,
                    child: InkWell(
                      onTap: () => context.push(
                        '/profiles/$_resolvedCounterpartProfileId',
                      ),
                      borderRadius: BorderRadius.circular(999),
                      child: CircleAvatar(
                        radius: 22,
                        backgroundColor: Theme.of(
                          context,
                        ).colorScheme.primaryContainer,
                        backgroundImage: _counterpartAvatarUrl.isNotEmpty
                            ? NetworkImage(_counterpartAvatarUrl)
                            : null,
                        child: _counterpartAvatarUrl.isEmpty
                            ? const Icon(Icons.person_rounded)
                            : null,
                      ),
                    ),
                  ),
                ],
              ],
            ),
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
                      controller: _messagesScrollController,
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
            _buildComposer(l10n),
          ],
        ),
      ),
    );
  }

  String _sanitizeTitle(
    String rawTitle, {
    required String counterpartProfileId,
  }) {
    final trimmed = rawTitle.trim();
    if (trimmed.isEmpty) {
      return '';
    }
    if (trimmed == counterpartProfileId || _uuidPattern.hasMatch(trimmed)) {
      return '';
    }
    final chatWithUuidMatch = _chatWithUuidPattern.firstMatch(trimmed);
    if (chatWithUuidMatch != null &&
        _uuidPattern.hasMatch(chatWithUuidMatch.group(1)!)) {
      return '';
    }
    return trimmed;
  }
}
