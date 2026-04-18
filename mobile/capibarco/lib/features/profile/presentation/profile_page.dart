import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/localization/app_localizations.dart';
import '../../../shared/presentation/page_shell.dart';
import '../../../shared/presentation/section_header.dart';
import '../../../shared/presentation/soft_card.dart';
import '../../../shared/presentation/status_view.dart';
import '../../auth/presentation/auth_controller.dart';
import 'profile_controller.dart';

class ProfilePage extends ConsumerStatefulWidget {
  const ProfilePage({super.key});

  @override
  ConsumerState<ProfilePage> createState() => _ProfilePageState();
}

class _ProfilePageState extends ConsumerState<ProfilePage> {
  @override
  void initState() {
    super.initState();
    Future<void>.microtask(
      () => ref.read(profileControllerProvider.notifier).load(),
    );
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final state = ref.watch(profileControllerProvider);
    final authState = ref.watch(authControllerProvider);

    return Scaffold(
      body: PageShell(
        child: ListView(
          children: <Widget>[
            SectionHeader(
              title: l10n.profile,
              subtitle: authState.session?.user.email ?? 'Profile workspace',
            ),
            const SizedBox(height: 16),
            if (state.isLoading && state.profile == null)
              StatusView.loading(message: l10n.loading)
            else if (state.errorMessage != null && state.profile == null)
              StatusView.message(
                message: state.errorMessage!,
                icon: Icons.error_outline_rounded,
              )
            else if (state.profile != null)
              SoftCard(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: <Widget>[
                    Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: <Widget>[
                        CircleAvatar(
                          radius: 32,
                          backgroundColor: Theme.of(
                            context,
                          ).colorScheme.primaryContainer,
                          backgroundImage: state.profile!.avatarUrl.isNotEmpty
                              ? NetworkImage(state.profile!.avatarUrl)
                              : null,
                          child: state.profile!.avatarUrl.isEmpty
                              ? const Icon(Icons.person_rounded)
                              : null,
                        ),
                        const SizedBox(width: 16),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: <Widget>[
                              Text(
                                state.profile!.displayName.isEmpty
                                    ? 'Unnamed profile'
                                    : state.profile!.displayName,
                                style: Theme.of(context).textTheme.headlineSmall
                                    ?.copyWith(fontWeight: FontWeight.w900),
                              ),
                              const SizedBox(height: 6),
                              Text(
                                '${state.profile!.typeLabel}${state.profile!.city.isNotEmpty ? ' · ${state.profile!.city}' : ''}',
                              ),
                              const SizedBox(height: 8),
                              Text(
                                state.profile!.bio.isEmpty
                                    ? 'Tell adopters and shelters a bit about yourself.'
                                    : state.profile!.bio,
                              ),
                              const SizedBox(height: 12),
                              Row(
                                children: <Widget>[
                                  const Icon(Icons.star_rounded, size: 18),
                                  const SizedBox(width: 6),
                                  Text(
                                    '${state.profile!.averageRating.toStringAsFixed(1)} (${state.profile!.reviewsCount} reviews)',
                                  ),
                                ],
                              ),
                            ],
                          ),
                        ),
                      ],
                    ),
                    if (state.infoMessage != null) ...<Widget>[
                      const SizedBox(height: 16),
                      Text(
                        state.infoMessage!,
                        style: TextStyle(
                          color: Theme.of(context).colorScheme.primary,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ],
                    if (state.errorMessage != null) ...<Widget>[
                      const SizedBox(height: 16),
                      Text(
                        state.errorMessage!,
                        style: TextStyle(
                          color: Theme.of(context).colorScheme.error,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ],
                    const SizedBox(height: 18),
                    FilledButton.tonalIcon(
                      onPressed: state.isSaving
                          ? null
                          : () => _showEditSheet(context, ref, state.profile!),
                      icon: const Icon(Icons.edit_rounded),
                      label: Text(l10n.editProfile),
                    ),
                    const SizedBox(height: 12),
                    FilledButton.icon(
                      onPressed: () => context.push('/profile/animals/new'),
                      icon: const Icon(Icons.pets_rounded),
                      label: Text(l10n.createPetCta),
                    ),
                    const SizedBox(height: 12),
                    OutlinedButton.icon(
                      onPressed: () =>
                          ref.read(authControllerProvider.notifier).logout(),
                      icon: const Icon(Icons.logout_rounded),
                      label: Text(l10n.signOut),
                    ),
                  ],
                ),
              ),
          ],
        ),
      ),
    );
  }

  Future<void> _showEditSheet(
    BuildContext context,
    WidgetRef ref,
    profile,
  ) async {
    final l10n = AppLocalizations.of(context);
    final nameController = TextEditingController(text: profile.displayName);
    final bioController = TextEditingController(text: profile.bio);
    final cityController = TextEditingController(text: profile.city);
    var profileType = profile.typeCode;

    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Theme.of(context).colorScheme.surfaceContainerLowest,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(32)),
      ),
      builder: (context) {
        return Padding(
          padding: EdgeInsets.fromLTRB(
            20,
            24,
            20,
            MediaQuery.of(context).viewInsets.bottom + 20,
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: <Widget>[
              TextField(
                controller: nameController,
                decoration: const InputDecoration(labelText: 'Display name'),
              ),
              const SizedBox(height: 12),
              StatefulBuilder(
                builder: (context, setModalState) {
                  return DropdownButtonFormField<String>(
                    initialValue: profileType,
                    decoration: InputDecoration(labelText: l10n.profileType),
                    items: <DropdownMenuItem<String>>[
                      DropdownMenuItem(
                        value: 'PROFILE_TYPE_USER',
                        child: Text(l10n.userProfile),
                      ),
                      DropdownMenuItem(
                        value: 'PROFILE_TYPE_SHELTER',
                        child: Text(l10n.shelterProfile),
                      ),
                      DropdownMenuItem(
                        value: 'PROFILE_TYPE_KENNEL',
                        child: Text(l10n.kennelProfile),
                      ),
                    ],
                    onChanged: (value) {
                      if (value == null) {
                        return;
                      }
                      setModalState(() => profileType = value);
                    },
                  );
                },
              ),
              const SizedBox(height: 12),
              TextField(
                controller: cityController,
                decoration: InputDecoration(labelText: l10n.city),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: bioController,
                minLines: 3,
                maxLines: 5,
                decoration: InputDecoration(labelText: l10n.bio),
              ),
              const SizedBox(height: 16),
              FilledButton(
                onPressed: () async {
                  await ref
                      .read(profileControllerProvider.notifier)
                      .updateProfile(
                        displayName: nameController.text.trim(),
                        bio: bioController.text.trim(),
                        city: cityController.text.trim(),
                        profileType: profileType,
                        infoMessage: profileType == 'PROFILE_TYPE_KENNEL'
                            ? l10n.createKennelProfile
                            : l10n.profileUpdated,
                      );
                  if (context.mounted) {
                    Navigator.of(context).pop();
                  }
                },
                child: Text(l10n.save),
              ),
            ],
          ),
        );
      },
    );

    nameController.dispose();
    bioController.dispose();
    cityController.dispose();
  }
}
