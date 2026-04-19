import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/localization/app_localizations.dart';
import '../../../bootstrap/providers.dart';
import '../../../core/config/environment.dart';
import '../../../core/network/network_providers.dart';
import '../../../core/network/rest_service_client.dart';
import '../../../features/feed/presentation/feed_controller.dart';
import '../../../shared/presentation/animal_details_sheet.dart';
import '../../../shared/presentation/page_shell.dart';
import '../../../shared/presentation/section_header.dart';
import '../../../shared/presentation/soft_card.dart';
import '../../../shared/presentation/status_view.dart';
import '../../auth/presentation/auth_controller.dart';
import '../../chat/presentation/chat_repository_provider.dart';
import '../data/api/profile_api_client.dart';
import '../data/datasources/public_profile_remote_data_source.dart';
import '../data/repositories/public_profile_repository_impl.dart';
import '../domain/entities/public_profile_detail.dart';
import 'profile_controller.dart';

final publicProfileRepositoryProvider = Provider<PublicProfileRepositoryImpl>((
  ref,
) {
  final environment = ref.watch(appEnvironmentProvider);
  return PublicProfileRepositoryImpl(
    remoteDataSource: PublicProfileRemoteDataSource(
      ProfileApiClient(
        RestServiceClient(
          dio: ref.watch(authenticatedDioProvider),
          config: environment.service(ServiceKind.profiles),
        ),
      ),
    ),
    errorMapper: ref.watch(errorMapperProvider),
  );
});

class PublicProfilePage extends ConsumerStatefulWidget {
  const PublicProfilePage({required this.profileId, super.key});

  final String profileId;

  @override
  ConsumerState<PublicProfilePage> createState() => _PublicProfilePageState();
}

class _PublicProfilePageState extends ConsumerState<PublicProfilePage> {
  PublicProfileDetailEntity? _detail;
  bool _isLoading = true;
  bool _isStartingChat = false;
  bool _isSubmittingReview = false;
  String? _errorMessage;

  @override
  void initState() {
    super.initState();
    Future<void>.microtask(_loadProfile);
  }

  Future<void> _loadProfile() async {
    setState(() {
      _isLoading = true;
      _errorMessage = null;
    });
    try {
      final detail = await ref
          .read(publicProfileRepositoryProvider)
          .getDetail(widget.profileId);
      if (!mounted) {
        return;
      }
      setState(() {
        _detail = detail;
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

  Future<void> _startChat() async {
    if (_detail == null || _isStartingChat) {
      return;
    }

    final currentProfileId =
        ref.read(authControllerProvider).session?.user.id ?? '';
    if (currentProfileId.isEmpty || currentProfileId == widget.profileId) {
      return;
    }

    setState(() => _isStartingChat = true);
    try {
      final ids = <String>[currentProfileId, widget.profileId]..sort();
      final conversation = await ref
          .read(chatRepositoryProvider)
          .createConversation(
            targetProfileId: widget.profileId,
            idempotencyKey: 'direct:${ids.join(':')}',
          );
      if (!mounted) {
        return;
      }
      setState(() => _isStartingChat = false);
      final destination = Uri(
        path: '/chat/${conversation.id}',
        queryParameters: <String, String>{
          'return_to': '/profiles/${widget.profileId}',
          'profile_id': widget.profileId,
          if (_detail!.profile.displayName.trim().isNotEmpty)
            'title': _detail!.profile.displayName.trim(),
        },
      ).toString();
      context.push(destination);
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() => _isStartingChat = false);
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('Response sent')));
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final currentProfileId =
        ref.watch(profileControllerProvider).profile?.id ??
        ref.watch(authControllerProvider).session?.user.id ??
        '';
    final isOwnProfile =
        currentProfileId.isNotEmpty && currentProfileId == widget.profileId;

    return Scaffold(
      body: PageShell(
        child: ListView(
          children: <Widget>[
            SectionHeader(
              title: _detail?.profile.displayName ?? l10n.profile,
              subtitle: l10n.publicProfileSubtitle,
            ),
            const SizedBox(height: 16),
            if (_isLoading && _detail == null)
              StatusView.loading(message: l10n.loading)
            else if (_errorMessage != null && _detail == null)
              StatusView.message(
                message: _errorMessage!,
                icon: Icons.error_outline_rounded,
                action: FilledButton(
                  onPressed: _loadProfile,
                  child: Text(l10n.retry),
                ),
              )
            else if (_detail != null) ...<Widget>[
              SoftCard(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: <Widget>[
                    Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: <Widget>[
                        CircleAvatar(
                          radius: 34,
                          backgroundColor: Theme.of(
                            context,
                          ).colorScheme.primaryContainer,
                          backgroundImage: _detail!.profile.avatarUrl.isNotEmpty
                              ? NetworkImage(_detail!.profile.avatarUrl)
                              : null,
                          child: _detail!.profile.avatarUrl.isEmpty
                              ? const Icon(Icons.person_rounded)
                              : null,
                        ),
                        const SizedBox(width: 16),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: <Widget>[
                              Text(
                                _detail!.profile.displayName.isEmpty
                                    ? 'Unnamed profile'
                                    : _detail!.profile.displayName,
                                style: Theme.of(context).textTheme.headlineSmall
                                    ?.copyWith(fontWeight: FontWeight.w900),
                              ),
                              const SizedBox(height: 6),
                              Text(
                                '${_detail!.profile.typeLabel}${_detail!.profile.city.isNotEmpty ? ' В· ${_detail!.profile.city}' : ''}',
                              ),
                              const SizedBox(height: 10),
                              Row(
                                children: <Widget>[
                                  const Icon(Icons.star_rounded, size: 18),
                                  const SizedBox(width: 6),
                                  Text(
                                    '${_detail!.profile.averageRating.toStringAsFixed(1)} (${_detail!.profile.reviewsCount} ${l10n.reviewsLabel})',
                                  ),
                                ],
                              ),
                            ],
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 16),
                    Text(
                      l10n.aboutProfile,
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.w800,
                      ),
                    ),
                    const SizedBox(height: 8),
                    Text(
                      _detail!.profile.bio.isEmpty
                          ? 'This profile has not added a bio yet.'
                          : _detail!.profile.bio,
                    ),
                    const SizedBox(height: 18),
                    Text(
                      l10n.profileActions,
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.w800,
                      ),
                    ),
                    const SizedBox(height: 10),
                    FilledButton.icon(
                      onPressed: isOwnProfile
                          ? null
                          : (_isStartingChat ? null : _startChat),
                      icon: _isStartingChat
                          ? const SizedBox(
                              width: 16,
                              height: 16,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            )
                          : const Icon(Icons.chat_bubble_rounded),
                      label: Text(l10n.startChat),
                    ),
                    const SizedBox(height: 10),
                    FilledButton.tonalIcon(
                      onPressed: isOwnProfile || _isSubmittingReview
                          ? null
                          : () => _showReviewSheet(context),
                      icon: _isSubmittingReview
                          ? const SizedBox(
                              width: 16,
                              height: 16,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            )
                          : const Icon(Icons.rate_review_rounded),
                      label: const Text('Leave comment'),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 16),
              SectionHeader(
                title: l10n.comments,
                subtitle: '${_detail!.reviews.length} ${l10n.reviewsLabel}',
              ),
              const SizedBox(height: 12),
              if (_detail!.reviews.isEmpty)
                SoftCard(child: Text(l10n.noComments))
              else
                ..._detail!.reviews.map(
                  (review) => Padding(
                    padding: const EdgeInsets.only(bottom: 12),
                    child: SoftCard(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: <Widget>[
                          Row(
                            children: <Widget>[
                              const Icon(Icons.star_rounded, size: 18),
                              const SizedBox(width: 6),
                              Text('${review.rating}/5'),
                              const Spacer(),
                              Text(
                                'Reviewer',
                                style: Theme.of(context).textTheme.labelMedium,
                              ),
                            ],
                          ),
                          const SizedBox(height: 10),
                          Text(
                            review.text.isEmpty
                                ? 'No review text.'
                                : review.text,
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              const SizedBox(height: 16),
              SectionHeader(
                title: l10n.profilePets,
                subtitle:
                    '${_detail!.animals.length} ${l10n.publishedPets.toLowerCase()}',
              ),
              const SizedBox(height: 12),
              if (_detail!.animals.isEmpty)
                SoftCard(child: Text(l10n.noProfilePets))
              else
                ..._detail!.animals.map(
                  (animal) => Padding(
                    padding: const EdgeInsets.only(bottom: 12),
                    child: InkWell(
                      borderRadius: BorderRadius.circular(28),
                      onTap: () => _showAnimalDetails(
                        context,
                        ref,
                        animal,
                        isOwnProfile,
                      ),
                      child: SoftCard(
                        child: Row(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: <Widget>[
                            ClipRRect(
                              borderRadius: BorderRadius.circular(22),
                              child: SizedBox(
                                width: 92,
                                height: 92,
                                child: animal.photoUrl.isEmpty
                                    ? DecoratedBox(
                                        decoration: BoxDecoration(
                                          color: Theme.of(
                                            context,
                                          ).colorScheme.primaryContainer,
                                        ),
                                        child: const Icon(Icons.pets_rounded),
                                      )
                                    : Image.network(
                                        animal.photoUrl,
                                        fit: BoxFit.cover,
                                      ),
                              ),
                            ),
                            const SizedBox(width: 14),
                            Expanded(
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: <Widget>[
                                  Text(
                                    animal.name,
                                    style: Theme.of(context)
                                        .textTheme
                                        .titleMedium
                                        ?.copyWith(fontWeight: FontWeight.w800),
                                  ),
                                  const SizedBox(height: 4),
                                  Text(
                                    [
                                      animal.speciesLabel,
                                      if (animal.breed.isNotEmpty) animal.breed,
                                      if (animal.city.isNotEmpty) animal.city,
                                    ].join(' В· '),
                                  ),
                                  const SizedBox(height: 8),
                                  DecoratedBox(
                                    decoration: BoxDecoration(
                                      color: Theme.of(
                                        context,
                                      ).colorScheme.secondaryContainer,
                                      borderRadius: BorderRadius.circular(999),
                                    ),
                                    child: Padding(
                                      padding: const EdgeInsets.symmetric(
                                        horizontal: 12,
                                        vertical: 6,
                                      ),
                                      child: Text(
                                        animal.statusLabel.isEmpty
                                            ? 'available'
                                            : animal.statusLabel,
                                      ),
                                    ),
                                  ),
                                ],
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                  ),
                ),
            ],
          ],
        ),
      ),
    );
  }

  Future<void> _showAnimalDetails(
    BuildContext context,
    WidgetRef ref,
    animal,
    bool isOwnProfile,
  ) {
    final subtitleParts = <String>[
      animal.speciesLabel,
      if (animal.breed.isNotEmpty) animal.breed,
      if (animal.city.isNotEmpty) animal.city,
    ];

    return showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Theme.of(context).colorScheme.surfaceContainerLowest,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
      ),
      builder: (sheetContext) => AnimalDetailsSheet(
        name: animal.name,
        subtitle: subtitleParts.join(' В· '),
        description: animal.description,
        photoUrl: animal.photoUrl,
        statusLabel: animal.statusLabel,
        respondLabel: isOwnProfile ? 'Own card' : 'Respond',
        onRespond: isOwnProfile
            ? null
            : () async {
                await ref
                    .read(feedRepositoryProvider)
                    .swipeAnimal(
                      animalId: animal.id,
                      ownerProfileId: widget.profileId,
                      liked: true,
                      feedCardId: '',
                      feedSessionId: '',
                    );
                if (sheetContext.mounted) {
                  Navigator.of(sheetContext).pop();
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('Response sent')),
                  );
                }
              },
      ),
    );
  }

  Future<void> _showReviewSheet(BuildContext context) async {
    final textController = TextEditingController();
    var rating = 5;

    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Theme.of(context).colorScheme.surfaceContainerLowest,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
      ),
      builder: (sheetContext) => StatefulBuilder(
        builder: (modalContext, setModalState) => Padding(
          padding: EdgeInsets.fromLTRB(
            20,
            20,
            20,
            MediaQuery.of(modalContext).viewInsets.bottom + 20,
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              Text(
                'Leave comment',
                style: Theme.of(
                  modalContext,
                ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w900),
              ),
              const SizedBox(height: 14),
              DropdownButtonFormField<int>(
                initialValue: rating,
                decoration: const InputDecoration(labelText: 'Rating'),
                items: const <DropdownMenuItem<int>>[
                  DropdownMenuItem(value: 5, child: Text('5')),
                  DropdownMenuItem(value: 4, child: Text('4')),
                  DropdownMenuItem(value: 3, child: Text('3')),
                  DropdownMenuItem(value: 2, child: Text('2')),
                  DropdownMenuItem(value: 1, child: Text('1')),
                ],
                onChanged: (value) =>
                    setModalState(() => rating = value ?? rating),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: textController,
                minLines: 3,
                maxLines: 5,
                decoration: const InputDecoration(
                  labelText: 'Comment',
                  hintText: 'Write a few words',
                ),
              ),
              const SizedBox(height: 16),
              Row(
                children: <Widget>[
                  Expanded(
                    child: OutlinedButton(
                      onPressed: _isSubmittingReview
                          ? null
                          : () => Navigator.of(sheetContext).pop(),
                      child: const Text('Close'),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: FilledButton(
                      onPressed: _isSubmittingReview
                          ? null
                          : () async {
                              setState(() => _isSubmittingReview = true);
                              try {
                                await ref
                                    .read(publicProfileRepositoryProvider)
                                    .createReview(
                                      profileId: widget.profileId,
                                      rating: rating,
                                      text: textController.text.trim(),
                                    );
                                await _loadProfile();
                                if (sheetContext.mounted) {
                                  Navigator.of(sheetContext).pop();
                                }
                              } catch (error) {
                                if (modalContext.mounted) {
                                  ScaffoldMessenger.of(
                                    modalContext,
                                  ).showSnackBar(
                                    SnackBar(content: Text(error.toString())),
                                  );
                                }
                              } finally {
                                if (mounted) {
                                  setState(() => _isSubmittingReview = false);
                                }
                              }
                            },
                      child: _isSubmittingReview
                          ? const SizedBox(
                              width: 18,
                              height: 18,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            )
                          : const Text('Send'),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );

    textController.dispose();
  }
}
