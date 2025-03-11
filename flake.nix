{
  description = "Development Shell";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-24.11";
    flake-utils.url = "github:numtide/flake-utils";        # Useful for multi-platform support
  };

  outputs = { self, nixpkgs, flake-utils }: flake-utils.lib.eachSystem [ "x86_64-linux" "aarch64-linux" ] (system: let
    pkgs = import nixpkgs {
      inherit system;
    };
  in {
    devShell = pkgs.mkShell {
      buildInputs = with pkgs; [
        entr
        go
      ];

      shellHook = ''
        # Set your GOPATH or GOROOT if needed, otherwise Go should work out of the box
        export GOPATH=$HOME/go
        # export GOROOT=${pkgs.go}
        export PATH=$PATH:$GOPATH/bin:$GOROOT/bin
        echo "Development shell"
        exec $SHELL # use user shell
      '';
    };
  });
}
